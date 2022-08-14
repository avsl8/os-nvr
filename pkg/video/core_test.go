package video

import (
	"context"
	"sync"
	"testing"

	"nvr/pkg/log"

	"github.com/stretchr/testify/require"
)

type cancelFunc func()

func newTestServer(t *testing.T) (*Server, cancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}

	logger := log.NewMockLogger()
	if err := logger.Start(ctx); err != nil {
		require.NoError(t, err)
	}

	go logger.LogToStdout(ctx)

	p := NewServer(logger, &wg, 8554, 8888)
	if err := p.Start(ctx); err != nil {
		require.NoError(t, err)
	}

	cancelFunc := func() {
		cancel()
		wg.Wait()
	}

	return p, cancelFunc
}

func TestNewPath(t *testing.T) {
	p, cancel := newTestServer(t)
	defer cancel()

	c := PathConf{MonitorID: "x"}

	actual, cancel2, err := p.NewPath("mypath", c)
	require.NoError(t, err)
	actual.StreamInfo = nil
	actual.WaitForNewHLSsegment = nil

	expected := ServerPath{
		HlsAddress:   "http://127.0.0.1:8888/hls/mypath/index.m3u8",
		RtspAddress:  "rtsp://127.0.0.1:8554/mypath",
		RtspProtocol: "tcp",
	}
	require.Equal(t, expected, *actual)

	require.True(t, p.PathExist("mypath"))
	cancel2()

	require.False(t, p.PathExist("mypath"))
}
