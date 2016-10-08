#!/usr/bin/make -f

SHELL=/bin/bash

all:
	# go get -u
	go-bindata www/...
	GOARCH=amd64 GOOS=linux go build -o xethru-web-linux-64 .
	gzip --best xethru-web-linux-64
	GOARCH=amd64 GOOS=windows go build -o xethru-web-windows-64.exe .
	gzip --best xethru-web-windows-64.exe
	GOARCH=amd64 GOOS=darwin go build -o xethru-web-mac-64 .
	gzip --best xethru-web-mac-64
	GOARCH=386 GOOS=linux go build -o xethru-web-linux-32 .
	gzip --best xethru-web-linux-32
	GOARCH=386 GOOS=windows go build -o xethru-web-windows-32.exe .
	gzip --best xethru-web-windows-32.exe


clean:
	rm xethru-web-*
