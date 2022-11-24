package main

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"time"

	"go4.org/netipx"
	"golang.org/x/sync/errgroup"
)

const (
	concurrency int           = 32
	rawPrefix   string        = "2403:5807:59::1/64"
	sleepTime   time.Duration = 10 * time.Second
)

func newResolver(nameserver string, timeout time.Duration) *net.Resolver {
	return &net.Resolver{
		PreferGo:     true,
		StrictErrors: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: timeout,
			}
			return d.DialContext(ctx, network, nameserver)
		},
	}
}

func main() {
	workChan := make(chan netip.Addr, concurrency)
	g := new(errgroup.Group)
	resolver := newResolver("ip6-localhost:53", 1*time.Second)
	for w := 0; w < concurrency; w++ {
		workerID := w
		g.Go(func() error {
			ctx := context.Background()
			for job := range workChan {
				res, err := resolver.LookupAddr(ctx, job.String())
				if err != nil {
					fmt.Printf("WorkerID: %d Err: %s\n", workerID, err)
				}
				fmt.Println(workerID, res[0])
			}
			return nil
		})
	}

	// parse the raw string prefix into a netip Prefix
	prefix, err := netip.ParsePrefix(rawPrefix)
	if err != nil {
		panic(err)
	}
	// find the last IP address within the prefix
	last := netipx.PrefixLastIP(prefix)

	// create a range of IP addresses that we can iterate through to do the DNS
	// lookups with.
	p := netipx.MustParseIPRange(fmt.Sprintf("%s-%s", prefix.Addr(), last))
	time.Sleep(sleepTime)

	// process the IP addresses, passing them to the buffered channel for processing
	// by the worker group.
	for i := p.From(); i.Less(p.To()); i.Next() {
		workChan <- i
		i = i.Next()
	}
}
