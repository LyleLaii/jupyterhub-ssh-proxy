package sshproxy

import (
	"fmt"
	"io"
	"net"

	"jupyterhub-ssh-proxy/jupyterhubserver"

	log "github.com/lylelaii/golang_utils/logger/v1"
	"golang.org/x/crypto/ssh"
)

const MODULERNAME = "ssh-proxy"

type SshProxyServer struct {
	addr     string
	host_key ssh.Signer
	listener net.Listener
	jhserver *jupyterhubserver.JupyterHubServer
	logger   log.Logger
}

func NewSshProxyServer(addr string, host_key ssh.Signer, jhserver *jupyterhubserver.JupyterHubServer, logger log.Logger) *SshProxyServer {
	return &SshProxyServer{addr: addr, host_key: host_key, jhserver: jhserver, logger: logger}
}

func (s *SshProxyServer) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		s.logger.Info(MODULERNAME, fmt.Sprintf("net.Listen failed: %v", err))
		return err
	}
	s.listener = listener

	defer s.Close()

	var singleusers map[string]*jupyterhubserver.SingleUser = make(map[string]*jupyterhubserver.SingleUser)

	for {

		serverConf := &ssh.ServerConfig{
			PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
				// s.logger.Info(MODULERNAME, fmt.Sprintf("Login attempt: %s, user %s password: %s", c.RemoteAddr(), c.User(), string(pass)))
				s.logger.Info(MODULERNAME, fmt.Sprintf("Login attempt: %s, user %s", c.RemoteAddr(), c.User()))
				// TODO: support publickey
				clientConfig := &ssh.ClientConfig{User: s.jhserver.GetConnUser(),
					Auth: []ssh.AuthMethod{
						ssh.Password(s.jhserver.GetConnPasswd()),
					},
					HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
						return nil
					},
					BannerCallback: ssh.BannerDisplayStderr(),
				}

				if !s.jhserver.CheckUser(c.User(), string(pass)) {
					s.logger.Warn(MODULERNAME, "CheckUser return false.")
					return nil, fmt.Errorf("permission denied")
				}

				server := singleusers[c.User()].GetPodIP()

				if server == "" {
					s.logger.Error(MODULERNAME, "Did not find User Pod")
					return nil, fmt.Errorf("server not exist")
				}
				server = fmt.Sprintf("%s:%s", server, s.jhserver.GetSshPort())
				s.logger.Info(MODULERNAME, fmt.Sprintf("user: %s prepare connection to %s", c.User(), server))
				client, err := ssh.Dial("tcp", server, clientConfig)

				singleusers[c.User()].UpdateClient(client)
				return nil, err
			},
			BannerCallback: func(c ssh.ConnMetadata) string {
				// podName := s.jhserver.CheckPod(c.User())
				podIP := s.jhserver.GetPodIP(c.User())
				singleusers[c.User()] = jupyterhubserver.NewSingleUser(c.User(), "", podIP, &ssh.Client{})

				message := "Welcome to JupyterHub SSH Client! \nNow Check Pod status... \n"
				if podIP == "" {
					message += "Did not find pod, please make sure user enviroment is running! \n"
				} else {
					message += fmt.Sprintf("Pod %s is running, use token to login, have fun! \n", singleusers[c.User()].GetPodName())
				}

				return message
			},
		}

		serverConf.AddHostKey(s.host_key)

		conn, err := listener.Accept()
		if err != nil {
			s.logger.Error(MODULERNAME, fmt.Sprintf("listen.Accept failed: %v", err))
			return err
		}

		sshconnprxy := &SshConnProxy{Conn: conn,
			callbackFn: func(c ssh.ConnMetadata) (*ssh.Client, error) {

				client := singleusers[c.User()].GetClient()
				s.logger.Info(MODULERNAME, fmt.Sprintf("Connection accepted from: %s", c.RemoteAddr()))

				return client, err
			},
			wrapFn: func(c ssh.ConnMetadata, r io.ReadCloser) (io.ReadCloser, error) {
				return NewTypeWriterReadCloser(r), nil
			},
			closeFn: func(c ssh.ConnMetadata) error {
				s.logger.Info(MODULERNAME, "Connection closed.")
				return nil
			},
			logger: s.logger}

		go func() {
			if err := sshconnprxy.proxy(serverConf); err != nil {
				s.logger.Error(MODULERNAME, fmt.Sprintf("Error occured while serving %s\n", err))
				return
			}

			s.logger.Info(MODULERNAME, "Connection closed.")
		}()
	}

}

func (s *SshProxyServer) Close() error {
	return s.listener.Close()
}
