package sshproxy

import (
	"fmt"
	"io"
	"net"

	log "github.com/lylelaii/golang_utils/logger/v1"
	"golang.org/x/crypto/ssh"
)

type SshConnProxy struct {
	net.Conn
	callbackFn func(c ssh.ConnMetadata) (*ssh.Client, error)
	wrapFn     func(c ssh.ConnMetadata, r io.ReadCloser) (io.ReadCloser, error)
	closeFn    func(c ssh.ConnMetadata) error
	logger     log.Logger
}

func (p *SshConnProxy) proxy(serverConf *ssh.ServerConfig) error {
	serverConn, chans, reqs, err := ssh.NewServerConn(p, serverConf)
	if err != nil {
		p.logger.Error(MODULERNAME, "failed to handshake")
		return (err)
	}

	defer serverConn.Close()

	clientConn, err := p.callbackFn(serverConn)
	if err != nil {
		p.logger.Error(MODULERNAME, fmt.Sprintf("failed to %s", err.Error()))
		return (err)
	}

	defer clientConn.Close()

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {

		channel2, requests2, err2 := clientConn.OpenChannel(newChannel.ChannelType(), newChannel.ExtraData())
		if err2 != nil {
			p.logger.Error(MODULERNAME, fmt.Sprintf("Could not accept client channel: %s", err.Error()))

			return err
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			p.logger.Error(MODULERNAME, fmt.Sprintf("Could not accept server channel: %s", err.Error()))

			return err
		}

		// connect requests
		go func() {
			p.logger.Info(MODULERNAME, "Waiting for request")

		r:
			for {
				var req *ssh.Request
				var dst ssh.Channel

				select {
				case req = <-requests:
					dst = channel2
				case req = <-requests2:
					dst = channel
				}

				// p.logger.Info(MODULERNAME, fmt.Sprintf("Request: %s %s %s %s\n", dst, req.Type, req.WantReply, req.Payload))

				b, err := dst.SendRequest(req.Type, req.WantReply, req.Payload)
				if err != nil {
					p.logger.Error(MODULERNAME, fmt.Sprintf("%s", err))

				}

				if req.WantReply {
					req.Reply(b, nil)
				}

				switch req.Type {
				case "exit-status":
					break r
				case "exec":
					// not supported (yet)
				default:
					p.logger.Info(MODULERNAME, req.Type)
				}
			}

			channel.Close()
			channel2.Close()
		}()

		// connect channels
		p.logger.Info(MODULERNAME, "Connecting channels.")

		var wrappedChannel io.ReadCloser = channel
		var wrappedChannel2 io.ReadCloser = channel2

		if p.wrapFn != nil {
			// wrappedChannel, err = p.wrapFn(channel)
			wrappedChannel2, _ = p.wrapFn(serverConn, channel2)
		}

		go io.Copy(channel2, wrappedChannel)
		go io.Copy(channel, wrappedChannel2)

		defer wrappedChannel.Close()
		defer wrappedChannel2.Close()
	}

	if p.closeFn != nil {
		p.closeFn(serverConn)
	}

	return nil
}
