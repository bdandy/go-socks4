package socks4

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"

	"golang.org/x/net/proxy"
)

const (
	socksVersion = 0x04
	socksConnect = 0x01
	socksBind    = 0x02

	socksIdent = "nobody@0.0.0.0"

	accessGranted        = 0x5a
	accessRejected       = 0x5b
	accessIdentdRequired = 0x5c
	accessIdentdFailed   = 0x5d
)

var (
	ErrWrongURL      = Error{"wrong server url", nil}
	ErrWrongConnType = Error{"no support for connections of type", nil}
	ErrConnFailed    = Error{"connection failed to socks4 server", nil}
	ErrHostUnknown   = Error{"unable to find IP address of host", nil}
	ErrSocksServer   = Error{"socks4 server error", nil}
	ErrConnRejected  = Error{"connection rejected", nil}
	ErrIdentRequired = Error{"socks4 server require valid identd", nil}
)

func init() {
	proxy.RegisterDialerType("socks4", func(u *url.URL, d proxy.Dialer) (proxy.Dialer, error) {
		return &socks4{url: u, dialer: d}, nil
	})
}

// Error is custom error type for better error handling
type Error struct {
	msg string
	err error
}

// Wrap wraps an error to custom error type and returns copy
func (e Error) Wrap(err error) Error {
	e.err = err
	return e
}

// Unwrap implements errors.Unwrap interface
func (e Error) Unwrap() error {
	return e.err
}

// Error implements error interface
func (e Error) Error() string {
	return fmt.Sprintf("%s %v", e.msg, e.err)
}

// Equal compares two custom errors if they same origin
func (e Error) Equal(err Error) bool {
	return e.msg == err.msg
}

type socks4 struct {
	url    *url.URL
	dialer proxy.Dialer
}

func (s *socks4) Dial(network, addr string) (c net.Conn, err error) {
	var buf []byte

	switch network {
	case "tcp", "tcp4":
	default:
		return nil, ErrWrongConnType.Wrap(err)
	}

	c, err = s.dialer.Dial(network, s.url.Host)
	if err != nil {
		return nil, ErrConnFailed.Wrap(err)
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, ErrWrongURL.Wrap(err)
	}

	ip, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		return nil, ErrHostUnknown.Wrap(err)
	}

	ip4 := ip.IP.To4()

	var bport [2]byte
	iport, _ := strconv.Atoi(port)
	binary.BigEndian.PutUint16(bport[:], uint16(iport))

	buf = []byte{socksVersion, socksConnect}
	buf = append(buf, bport[:]...)
	buf = append(buf, ip4...)
	buf = append(buf, socksIdent...)
	buf = append(buf, 0)

	i, err := c.Write(buf)
	if err != nil {
		return nil, ErrSocksServer.Wrap(err)
	}

	if l := len(buf); i != l {
		return nil, ErrSocksServer.Wrap(fmt.Errorf("written %d bytes, expected %d", i, l))
	}

	var resp [8]byte
	i, err = c.Read(resp[:])
	if err != nil && err != io.EOF {
		return nil, ErrSocksServer.Wrap(err)
	}

	if i != 8 {
		return nil, ErrSocksServer.Wrap(fmt.Errorf("read %d bytes, expected 8", i))
	}

	switch resp[1] {
	case accessGranted:
		return c, nil

	case accessIdentdRequired, accessIdentdFailed:
		return nil, ErrIdentRequired.Wrap(fmt.Errorf(strconv.FormatInt(int64(resp[1]), 16)))

	default:
		c.Close()
		return nil, ErrConnRejected.Wrap(fmt.Errorf(strconv.FormatInt(int64(resp[1]), 16)))
	}
}
