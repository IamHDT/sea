[![Docker Build Status](https://img.shields.io/docker/cloud/build/navigaid/sea.svg)](https://hub.docker.com/r/navigaid/sea) [![Docker Pulls](https://img.shields.io/docker/pulls/navigaid/sea.svg)](https://hub.docker.com/r/navigaid/sea) [![Docker Stars](https://img.shields.io/docker/stars/navigaid/sea.svg)](https://hub.docker.com/r/navigaid/sea)

Sea
===

clone of seashells.io

## Run in docker
```
docker run -it --name sea -p 1337:1337 -p 8000:8000 --rm navigaid/sea
```

## Run natively
```
git get github.com/navigaid/sea
$(go env GOPATH)/bin/sea
```

## Play
```
exec 3<>/dev/tcp/127.0.0.1/1337
jq . <(head -n1 <(jq -c --unbuffered . <(cat <&3)))
echo hello >&3
echo world >&3
htop >&3
neofetch >&3

neofetch | nc localhost 1337

neofetch > /dev/tcp/127.0.0.1/1337

cat <(neofetch) - | nc localhost 1337

htop | nc localhost 1337

clear | nc localhost 1337
```

## Refs
- https://stackoverflow.com/questions/32684119/exit-when-one-process-in-pipe-fails/53382807#53382807
- https://news.ycombinator.com/item?id=14739479
- https://github.com/anishathalye/seashells
- https://github.com/mthenw/frontail
- https://seashells.io/
