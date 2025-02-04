//go:build linux

package bindtodevice

import (
	"fmt"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrefixAddr(t *testing.T) {
	const (
		port    = 56789
		network = "tcp"
	)

	testCases := []struct {
		in   *prefixNetAddr
		want string
		name string
	}{{
		in: &prefixNetAddr{
			prefix:  testSubnetIPv4,
			network: network,
			port:    port,
		},
		want: fmt.Sprintf(
			"%s/%d",
			netip.AddrPortFrom(testSubnetIPv4.Addr(), port), testSubnetIPv4.Bits(),
		),
		name: "ipv4",
	}, {
		in: &prefixNetAddr{
			prefix:  testSubnetIPv6,
			network: network,
			port:    port,
		},
		want: fmt.Sprintf(
			"%s/%d",
			netip.AddrPortFrom(testSubnetIPv6.Addr(), port), testSubnetIPv6.Bits(),
		),
		name: "ipv6",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.in.String())
			assert.Equal(t, network, tc.in.Network())
		})
	}
}
