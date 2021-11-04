# go-socks4
Socks4 implementation for Go, compatible with net/proxy

## Features
- `socks4` 
- `socks4a`

## Usage

```go
package main

import (
	"errors"
	"net/url"

	"golang.org/x/net/proxy"
	"github.com/Bogdan-D/go-socks4"
)

func main() {
	addr, _ := url.Parse("socks4://ip:port")
	
	var socksErr socks4.Error

	dialer, err := proxy.FromURL(addr, proxy.Direct)
	// check error
	// and use your dialer as you with 
	c, err := dialer.Dial("tcp", "google.com:80")
	if err != nil && errors.As(err, &socksErr) {
		// handle error
	}
}
```


## Tests
If you know proxy server to connect to tests should be running like this
`
go test -socks4.url=socks4://localhost:8080
`




