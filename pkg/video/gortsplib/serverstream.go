package gortsplib

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/pion/rtp/v2"
)

type trackTypePayload struct {
	trackID int
	payload []byte
}

type serverStreamTrack struct {
	lastSequenceNumber uint32
	lastTimeRTP        uint32
	lastTimeNTP        int64
	lastSSRC           uint32
}

// ServerStream represents a single stream.
// This is in charge of
// - distributing the stream to each reader
// - gathering infos about the stream to generate SSRC and RTP-Info.
type ServerStream struct {
	tracks Tracks

	mutex          sync.RWMutex
	s              *Server
	readersUnicast map[*ServerSession]struct{}
	readers        map[*ServerSession]struct{}
	stTracks       []*serverStreamTrack
}

// NewServerStream allocates a ServerStream.
func NewServerStream(tracks Tracks) *ServerStream {
	tracks = tracks.clone()
	tracks.setControls()

	st := &ServerStream{
		tracks:         tracks,
		readersUnicast: make(map[*ServerSession]struct{}),
		readers:        make(map[*ServerSession]struct{}),
	}

	st.stTracks = make([]*serverStreamTrack, len(tracks))
	for i := range st.stTracks {
		st.stTracks[i] = &serverStreamTrack{}
	}

	return st
}

// Close closes a ServerStream.
func (st *ServerStream) Close() error {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	for ss := range st.readers {
		ss.Close()
	}

	st.readers = nil
	st.readersUnicast = nil

	return nil
}

// Tracks returns the tracks of the stream.
func (st *ServerStream) Tracks() Tracks {
	return st.tracks
}

func (st *ServerStream) ssrc(trackID int) uint32 {
	return atomic.LoadUint32(&st.stTracks[trackID].lastSSRC)
}

func (st *ServerStream) timestamp(trackID int) uint32 {
	lastTimeRTP := atomic.LoadUint32(&st.stTracks[trackID].lastTimeRTP)
	lastTimeNTP := atomic.LoadInt64(&st.stTracks[trackID].lastTimeNTP)

	if lastTimeRTP == 0 || lastTimeNTP == 0 {
		return 0
	}

	return uint32(uint64(lastTimeRTP) +
		uint64(
			time.Since(time.Unix(lastTimeNTP, 0)).
				Seconds()*float64(st.tracks[trackID].ClockRate()),
		),
	)
}

func (st *ServerStream) lastSequenceNumber(trackID int) uint16 {
	return uint16(atomic.LoadUint32(&st.stTracks[trackID].lastSequenceNumber))
}

func (st *ServerStream) readerAdd(ss *ServerSession) {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	if st.s == nil {
		st.s = ss.s
	}

	st.readers[ss] = struct{}{}
}

func (st *ServerStream) readerRemove(ss *ServerSession) {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	delete(st.readers, ss)
}

func (st *ServerStream) readerSetActive(ss *ServerSession) {
	st.mutex.Lock()
	st.readersUnicast[ss] = struct{}{}
	st.mutex.Unlock()
}

func (st *ServerStream) readerSetInactive(ss *ServerSession) {
	st.mutex.Lock()
	delete(st.readersUnicast, ss)
	st.mutex.Unlock()
}

// WritePacketRTP writes a RTP packet to all the readers of the stream.
func (st *ServerStream) WritePacketRTP(trackID int, pkt *rtp.Packet) {
	byts, err := pkt.Marshal()
	if err != nil {
		return
	}

	track := st.stTracks[trackID]
	now := time.Now()

	atomic.StoreUint32(&track.lastSequenceNumber,
		uint32(pkt.Header.SequenceNumber))
	atomic.StoreUint32(&track.lastTimeRTP, pkt.Header.Timestamp)
	atomic.StoreInt64(&track.lastTimeNTP, now.Unix())
	atomic.StoreUint32(&track.lastSSRC, pkt.Header.SSRC)

	st.mutex.RLock()
	defer st.mutex.RUnlock()

	// send unicast
	for r := range st.readersUnicast {
		r.writePacketRTP(trackID, byts)
	}
}