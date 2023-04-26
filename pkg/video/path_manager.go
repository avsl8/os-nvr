package video

import (
	"context"
	"errors"
	"fmt"
	"nvr/pkg/log"
	"nvr/pkg/video/gortsplib"
	"nvr/pkg/video/gortsplib/pkg/base"
	"nvr/pkg/video/hls"
	"sync"
)

type pathManagerHLSServer interface {
	pathSourceReady(*path, gortsplib.Tracks) (*HLSMuxer, error)
	pathSourceNotReady(pathName string)
	MuxerByPathName(ctx context.Context, pathName string) (*hls.Muxer, error)
}

type pathManager struct {
	wg  *sync.WaitGroup
	log log.ILogger
	mu  sync.Mutex

	hlsServer pathManagerHLSServer
	pathConfs map[string]*PathConf
	paths     map[string]*path
}

func newPathManager(
	wg *sync.WaitGroup,
	log log.ILogger,
	hlsServer pathManagerHLSServer,
) *pathManager {
	return &pathManager{
		wg:  wg,
		log: log,

		hlsServer: hlsServer,
		pathConfs: make(map[string]*PathConf),
		paths:     make(map[string]*path),
	}
}

// Errors.
var (
	ErrPathAlreadyExist = errors.New("path already exist")
	ErrPathNotExist     = errors.New("path not exist")
)

// AddPath add path to pathManager.
func (pm *pathManager) AddPath(
	ctx context.Context,
	name string,
	newConf PathConf,
) (HlsMuxerFunc, error) {
	debug := func(msg string) {
		pm.log.Log(log.Entry{
			Level: log.LevelDebug,
			Src:   "app",
			Msg:   fmt.Sprintf("%v: addPath: %v", name, msg),
		})
	}
	debug("lock")
	defer debug("unlock")
	pm.mu.Lock()
	defer pm.mu.Unlock()

	debug("A")
	err := newConf.CheckAndFillMissing(name)
	if err != nil {
		return nil, err
	}

	debug("B")
	if _, exist := pm.pathConfs[name]; exist {
		return nil, ErrPathAlreadyExist
	}

	config := &newConf

	// Add config.
	pm.pathConfs[name] = config

	// Add path.
	debug("C")
	pm.paths[name] = newPath(
		ctx,
		name,
		config,
		pm.wg,
		pm.hlsServer,
		pm.log,
	)

	hlsMuxer := func(ctx context.Context) (IHLSMuxer, error) {
		return pm.hlsServer.MuxerByPathName(ctx, name)
	}

	go func() {
		// Remove path.
		<-ctx.Done()

		debug("Remove Lock")
		defer debug("Remove Lock")
		pm.mu.Lock()
		defer pm.mu.Unlock()

		// Remove config.
		debug("Remove A")
		delete(pm.pathConfs, name)

		// Close and remove path.
		debug("Remove B")
		// Is this the deadlock.
		pm.paths[name].close()
		debug("Remove C")
		delete(pm.paths, name)
		debug("Remove D")
	}()
	debug("D")

	return hlsMuxer, nil
}

// Testing.
func (pm *pathManager) pathExist(name string) bool {
	debug := func(msg string) {
		pm.log.Log(log.Entry{
			Level: log.LevelDebug,
			Src:   "app",
			Msg:   fmt.Sprintf("%v: path exists: %v", name, msg),
		})
	}
	debug("lock")
	defer debug("unlock")
	pm.mu.Lock()
	defer pm.mu.Unlock()

	_, exist := pm.pathConfs[name]
	return exist
}

// describe is called by a rtsp reader.
func (pm *pathManager) onDescribe(
	pathName string,
) (*base.Response, *gortsplib.ServerStream, error) {
	debug := func(msg string) {
		pm.log.Log(log.Entry{
			Level: log.LevelDebug,
			Src:   "app",
			Msg:   fmt.Sprintf("%v: on describe: %v", pathName, msg),
		})
	}
	debug("lock")
	defer debug("unlock")
	pm.mu.Lock()
	defer pm.mu.Unlock()

	path, exist := pm.paths[pathName]
	if !exist {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, ErrPathNotExist
	}

	stream, err := path.streamGet()
	if err != nil {
		if errors.Is(err, ErrPathNoOnePublishing) {
			return &base.Response{
				StatusCode: base.StatusNotFound,
			}, nil, err
		}
		return &base.Response{
			StatusCode: base.StatusBadRequest,
		}, nil, err
	}

	return &base.Response{StatusCode: base.StatusOK}, stream.rtspStream, nil
}

// publisherAdd is called by a rtsp publisher.
func (pm *pathManager) publisherAdd(
	name string,
	session *rtspSession,
) (*path, error) {
	debug := func(msg string) {
		pm.log.Log(log.Entry{
			Level: log.LevelDebug,
			Src:   "app",
			Msg:   fmt.Sprintf("%v: publisher add: %v", name, msg),
		})
	}
	debug("lock")
	defer debug("unlock")
	pm.mu.Lock()
	defer pm.mu.Unlock()

	path, exist := pm.paths[name]
	if !exist {
		return nil, ErrPathNotExist
	}
	return path.publisherAdd(session)
}

// readerAdd is called by a rtsp reader.
func (pm *pathManager) readerAdd(
	name string,
	session *rtspSession,
) (*path, *stream, error) {
	debug := func(msg string) {
		pm.log.Log(log.Entry{
			Level: log.LevelDebug,
			Src:   "app",
			Msg:   fmt.Sprintf("%v: reader add: %v", name, msg),
		})
	}
	debug("lock")
	defer debug("unlock")
	pm.mu.Lock()
	defer pm.mu.Unlock()

	path, exist := pm.paths[name]
	if !exist {
		return nil, nil, ErrPathNotExist
	}
	return path.readerAdd(session)
}

func (pm *pathManager) pathLogfByName(name string) log.Func {
	debug := func(msg string) {
		pm.log.Log(log.Entry{
			Level: log.LevelDebug,
			Src:   "app",
			Msg:   fmt.Sprintf("%v: pathlogfbyname: %v", name, msg),
		})
	}
	debug("lock")
	defer debug("unlock")
	pm.mu.Lock()
	defer pm.mu.Unlock()

	path, exist := pm.paths[name]
	if exist {
		return path.logf
	}
	return nil
}
