package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var upgrader = websocket.Upgrader{}

func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	f := flag.NewFlagSet(args[0], flag.ExitOnError)
	var (
		path = f.String("c", "config.yaml", "Yaml configuration path")
	)
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	c, err := loadConfig(*path)
	if err != nil {
		return fmt.Errorf("Unable to load config: %v", err)
	}

	log.Printf("Server listening on addr: %s", c.Addr)

	return http.ListenAndServe(c.Addr,
		h2c.NewHandler(
			http.HandlerFunc(handler(c.HttpHosts, c.MaxTime)),
			&http2.Server{},
		),
	)

}

func handler(hosts []httpHost, maxtime uint) http.HandlerFunc {
	client := newHTTPClient(time.Duration(maxtime) * time.Second)
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if websocket.IsWebSocketUpgrade(r) {
			serveWS(w, r, client, hosts)
		} else {
			serveHTTP(w, r)
		}
	}
}

func serveWS(w http.ResponseWriter, r *http.Request, client *httpClient, hosts []httpHost) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer conn.Close()

	log.Printf("%s | upgraded to websocket", r.RemoteAddr)

	conn.SetPingHandler(func(msg string) error {
		r := make(chan error, len(hosts))
		for i := 0; i < cap(r); i++ {
			h := hosts[i]
			go func() {
				r <- client.curl(context.TODO(), h)
			}()
		}
		var success bool
		for i := 0; i < cap(r); i++ {
			err = <-r
			if err == nil {
				success = true
			}
		}
		if !success {
			return conn.WriteMessage(websocket.CloseMessage, []byte{})
		}
		return conn.WriteMessage(websocket.PongMessage, []byte(msg))
	})

	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		err = conn.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func serveHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(200)

	host, err := os.Hostname()
	if err == nil {
		fmt.Fprintf(w, "Request served by %s\n\n", host)
	} else {
		fmt.Fprintf(w, "Server hostname unknown: %s\n\n", err.Error())
	}

	writeRequest(w, r)
}

func writeRequest(w io.Writer, r *http.Request) {
	fmt.Fprintf(w, "%s %s %s\n", r.Proto, r.Method, r.URL)
	fmt.Fprintln(w, "")

	fmt.Fprintf(w, "Host: %s\n", r.Host)
	for key, values := range r.Header {
		for _, value := range values {
			fmt.Fprintf(w, "%s: %s\n", key, value)
		}
	}

	var body bytes.Buffer
	io.Copy(&body, r.Body) // nolint:errcheck

	if body.Len() > 0 {
		fmt.Fprintln(w, "")
		body.WriteTo(w) // nolint:errcheck
	}
}
