package toxy

import (
	"fmt"
	"net"
	"strings"
	"time"
	"xlog"
	"xmetric"

	. "github.com/stdrickforce/thriftgo/protocol"
	. "github.com/stdrickforce/thriftgo/transport"
)

type MultiplexedProcessor struct {
	pf   ProtocolFactory
	hmap map[string]*Handler
}

func (self *MultiplexedProcessor) Add(
	name string,
	handler *Handler,
) (err error) {
	self.hmap[name] = handler
	return
}

func (self *MultiplexedProcessor) parse_name(name string) (
	service, fname string,
) {
	segments := strings.SplitN(name, ":", 2)
	if len(segments) != 2 {
		err := NewProcessorError("fname format mismatch: %s", name)
		panic(err)
	}
	service, fname = segments[0], segments[1]
	return
}

func (self *MultiplexedProcessor) get_protocol(service string) Protocol {
	handler := self.hmap[service]
	if handler == nil {
		err := NewProcessorError("no handler has been set for: %s", service)
		panic(err)
	}

	proto, err := handler.GetProtocol()
	if err != nil {
		panic(err)
	}
	return proto
}

func (self *MultiplexedProcessor) handle(m *Messenger) bool {
	if shutdown > 0 {
		fast_reply_shutdown(m)
		return false
	}

	s_time := time.Now().UnixNano()

	name, seqid := read_header(m)

	service, fname := self.parse_name(name)
	xlog.Debug("%s: %s", service, fname)

	key := fmt.Sprintf("%s.%s", service, fname)
	defer func() {
		delta := int((time.Now().UnixNano() - s_time) / 1000000)
		xmetric.Timing("toxy", key, delta)
	}()
	xmetric.Count("toxy", key, 1)

	// NOTE fast reply ping requests.
	// is it neccessary?
	if fname == "ping" {
		fast_reply(m, seqid)
		return true
	}

	oprot := self.get_protocol(service)
	m.SetOutputProtocol(oprot)
	defer m.DelOutputProtocol()

	reply(m, fname, seqid)
	return true
}

func (self *MultiplexedProcessor) Process(conn net.Conn) {
	var itrans Transport
	itrans = NewTSocketConn(conn)
	itrans = NewTBufferedTransport(itrans)
	defer itrans.Close()

	protocol := self.pf.GetProtocol(itrans)
	m := NewMessenger(protocol)

	for {
		if ok := self.handle(m); !ok {
			break
		}
	}
}

func NewMultiplexedProcessor(pf ProtocolFactory) *MultiplexedProcessor {
	return &MultiplexedProcessor{
		pf:   pf,
		hmap: make(map[string]*Handler),
	}
}
