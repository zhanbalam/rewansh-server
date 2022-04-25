# rewansh-server

```
build -t rewansh-server .
docker run --rm -it -v `pwd`/config.yaml:/etc/rewansh-server/config.yaml -p 8080:8080 rewansh-server
```
