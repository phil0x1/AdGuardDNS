package forward_test

import (
	"context"
	"net/netip"
	"sync/atomic"
	"testing"

	"github.com/AdguardTeam/AdGuardDNS/internal/dnsserver"
	"github.com/AdguardTeam/AdGuardDNS/internal/dnsserver/dnsservertest"
	"github.com/AdguardTeam/AdGuardDNS/internal/dnsserver/forward"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_Refresh(t *testing.T) {
	var upstreamIsUp atomic.Bool
	var upstreamRequestsCount atomic.Int64

	defaultHandler := dnsservertest.DefaultHandler()

	// This handler writes an empty message if upstreamUp flag is false.
	handlerFunc := dnsserver.HandlerFunc(func(
		ctx context.Context,
		rw dnsserver.ResponseWriter,
		req *dns.Msg,
	) (err error) {
		upstreamRequestsCount.Add(1)

		nrw := dnsserver.NewNonWriterResponseWriter(rw.LocalAddr(), rw.RemoteAddr())
		err = defaultHandler.ServeDNS(ctx, nrw, req)
		if err != nil {
			return err
		}

		if !upstreamIsUp.Load() {
			return rw.WriteMsg(ctx, req, &dns.Msg{})
		}

		return rw.WriteMsg(ctx, req, nrw.Msg())
	})

	upstream, _ := dnsservertest.RunDNSServer(t, handlerFunc)
	fallback, _ := dnsservertest.RunDNSServer(t, defaultHandler)
	handler := forward.NewHandler(&forward.HandlerConfig{
		Address:               netip.MustParseAddrPort(upstream.LocalUDPAddr().String()),
		Network:               forward.NetworkAny,
		HealthcheckDomainTmpl: "${RANDOM}.upstream-check.example",
		FallbackAddresses: []netip.AddrPort{
			netip.MustParseAddrPort(fallback.LocalUDPAddr().String()),
		},
		Timeout: testTimeout,
		// Make sure that the handler routes queries back to the main upstream
		// immediately.
		HealthcheckBackoffDuration: 0,
	})

	req := dnsservertest.CreateMessage("example.org.", dns.TypeA)
	rw := dnsserver.NewNonWriterResponseWriter(fallback.LocalUDPAddr(), fallback.LocalUDPAddr())

	ctx := context.Background()

	err := handler.ServeDNS(newTimeoutCtx(t, ctx), rw, req)
	require.Error(t, err)
	assert.Equal(t, int64(2), upstreamRequestsCount.Load())

	err = handler.Refresh(newTimeoutCtx(t, ctx))
	require.Error(t, err)
	assert.Equal(t, int64(4), upstreamRequestsCount.Load())

	err = handler.ServeDNS(newTimeoutCtx(t, ctx), rw, req)
	require.NoError(t, err)
	assert.Equal(t, int64(4), upstreamRequestsCount.Load())

	// Now, set upstream up.
	upstreamIsUp.Store(true)

	err = handler.ServeDNS(newTimeoutCtx(t, ctx), rw, req)
	require.NoError(t, err)
	assert.Equal(t, int64(4), upstreamRequestsCount.Load())

	err = handler.Refresh(newTimeoutCtx(t, ctx))
	require.NoError(t, err)
	assert.Equal(t, int64(5), upstreamRequestsCount.Load())

	err = handler.ServeDNS(newTimeoutCtx(t, ctx), rw, req)
	require.NoError(t, err)
	assert.Equal(t, int64(6), upstreamRequestsCount.Load())
}
