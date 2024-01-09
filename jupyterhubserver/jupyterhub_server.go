package jupyterhubserver

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"

	log "github.com/lylelaii/golang_utils/logger/v1"
	requestes "github.com/lylelaii/golang_utils/requestes/v1"
	"golang.org/x/crypto/ssh"
)

const MODULENAME = "jupyterhubserver"

type JupyterHubServerConfig struct {
	Url                string `mapstructure:"url"`
	AdminToken         string `mapstructure:"admin_token"`
	ConnUser           string `mapstructure:"conn_user"`
	ConnPasswd         string `mapstructure:"conn_passwd"`
	SshPort            string `mapstructure:"ssh_port"`
	AuthorizedKeysPath string `mapstructure:"authorized_keys_path"`
	VerifyTLS          bool   `mapstructure:"verify_tls"`
}

type JupyterHubServer struct {
	url                string
	adminToken         string
	connUser           string
	connPasswd         string
	sshPort            string
	authorizedKeysPath string
	requestesClient    *requestes.RequestsClient
	logger             log.Logger
}

func NewJupyterHubServer(c JupyterHubServerConfig, logger log.Logger) *JupyterHubServer {
	requestesClinet, _ := requestes.New(requestes.RequestsConfig{VerifyTLS: c.VerifyTLS})
	return &JupyterHubServer{url: c.Url,
		adminToken:         c.AdminToken,
		connUser:           c.ConnUser,
		connPasswd:         c.ConnPasswd,
		sshPort:            c.SshPort,
		authorizedKeysPath: c.AuthorizedKeysPath,
		requestesClient:    requestesClinet,
		logger:             logger}
}

func (s *JupyterHubServer) GetConnUser() string {
	return s.connUser
}

func (s *JupyterHubServer) GetConnPasswd() string {
	return s.connPasswd
}

func (s *JupyterHubServer) GetSshPort() string {
	return s.sshPort
}

func (s *JupyterHubServer) queryUserInfo(username, password string) (bool, *UserInfo) {
	var headers map[string]string = make(map[string]string)
	headers["Authorization"] = fmt.Sprintf("token %s", password)
	// headers["Accept"] = "application/jupyterhub-pagination+json"

	uri := s.url + fmt.Sprintf("/users/%s", username)

	res, err := s.requestesClient.Get(uri, requestes.AddHeader(headers))
	if err != nil {
		s.logger.Warn(MODULENAME, fmt.Sprintf("queryUserInfo get err: %s", err.Error()))
		return false, &UserInfo{}
	}

	if res.StatusCode != http.StatusOK {
		s.logger.Info(MODULENAME, fmt.Sprintf("queryUserInfo get non 200 response code: %v", res.StatusCode))
		return false, &UserInfo{}
	}

	s.logger.Debug(MODULENAME, fmt.Sprintf("queryUserInfo rep: %v", res.Text()))
	var userInfo UserInfo
	err = res.BindJSON(&userInfo)

	if err != nil {
		s.logger.Error(MODULENAME, fmt.Sprintf("queryUserInfo bindJson get err: %v", err.Error()))
		return false, &UserInfo{}
	}

	s.logger.Info(MODULENAME, fmt.Sprintf("queryUserInfo %v : %v", username, &userInfo))

	return true, &userInfo

}

func (s *JupyterHubServer) queryUserRoute(username string) *UserRoute {
	var headers map[string]string = make(map[string]string)
	headers["Authorization"] = fmt.Sprintf("token %s", s.adminToken)
	// headers["Accept"] = "application/jupyterhub-pagination+json"

	uri := s.url + "/proxy"

	res, err := s.requestesClient.Get(uri, requestes.AddHeader(headers))
	if err != nil {
		s.logger.Warn(MODULENAME, fmt.Sprintf("queryUserRoute get err: %s", err.Error()))
		return &UserRoute{}
	}

	if res.StatusCode != http.StatusOK {
		s.logger.Info(MODULENAME, fmt.Sprintf("queryUserRoute get non 200 response code: %v", res.StatusCode))
		return &UserRoute{}
	}

	s.logger.Debug(MODULENAME, fmt.Sprintf("queryUserRoute rep: %v", res.Text()))
	var userRoutes UserRoutes
	err = res.BindJSON(&userRoutes)

	if err != nil {
		s.logger.Error(MODULENAME, fmt.Sprintf("queryUserRoute bindJson get err: %v", err.Error()))
		return &UserRoute{}
	}

	userIndex := fmt.Sprintf("/user/%s/", username)
	r := userRoutes[userIndex]

	s.logger.Debug(MODULENAME, fmt.Sprintf("queryUserRoute %v : %v", username, r))

	return &r

}

func (s *JupyterHubServer) GetPodIP(username string) string {
	userRoute := s.queryUserRoute(username)

	target := userRoute.Target
	ipReg := `((2(5[0-5]|[0-4]\d))|[0-1]?\d{1,2})(\.((2(5[0-5]|[0-4]\d))|[0-1]?\d{1,2})){3}`
	reg, _ := regexp.Compile(ipReg)
	podIP := reg.Find([]byte(target))

	s.logger.Debug(MODULENAME, fmt.Sprintf("GetPodIP %s : %s", username, string(podIP)))

	return string(podIP)
}

func (s *JupyterHubServer) CheckUser(username string, password string) bool {
	auth, _ := s.queryUserInfo(username, password)

	return auth
}

func (s *JupyterHubServer) CheckPod(username string) string {
	_, userInfo := s.queryUserInfo(username, s.adminToken)
	// fmt.Printf("%+v", userInfo)
	podName := userInfo.Servers.ServerDetail.State.PodName

	return podName
}

func (s *JupyterHubServer) GetUserAuthorizedKeys(podIP string) (map[string]bool, error) {
	config := &ssh.ClientConfig{
		User: s.connUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(s.connPasswd)},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	server := fmt.Sprintf("%s:%s", podIP, s.sshPort)
	client, err := ssh.Dial("tcp", server, config)
	if err != nil {
		s.logger.Error(MODULENAME, fmt.Sprintf("GetUserAuthorizedKey Create Client get err: %s", err.Error()))
		return make(map[string]bool), err
	}
	session, err := client.NewSession()
	if err != nil {
		s.logger.Error(MODULENAME, fmt.Sprintf("GetUserAuthorizedKey Create Session get err: %s", err.Error()))
		return make(map[string]bool), err
	}

	defer session.Close()

	authorizedKeysBytes, err := session.CombinedOutput(fmt.Sprintf("if [ -f %s ];then cat %s ; else echo ''; fi",
		s.authorizedKeysPath,
		s.authorizedKeysPath))
	if err != nil {
		s.logger.Error(MODULENAME, fmt.Sprintf("GetUserAuthorizedKey Run Command get err: %s", err.Error()))
		return make(map[string]bool), err
	}
	s.logger.Debug(MODULENAME, fmt.Sprintf("GetUserAuthorizedKey Run Command get %s", string(authorizedKeysBytes)))

	authorizedKeysMap := make(map[string]bool)
	transAuthorizedKeys := string(authorizedKeysBytes)
	transAuthorizedKeys = strings.Replace(transAuthorizedKeys, `\n`, "\n", -1)
	authorizedKeysBytes = []byte(transAuthorizedKeys)
	for len(authorizedKeysBytes) > 0 {
		publicKey, _, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
		if err != nil {
			s.logger.Error(MODULENAME, fmt.Sprintf("GetUserAuthorizedKey ParseAuthorizedKey get err: %s, origins: %s",
				err.Error(),
				string(authorizedKeysBytes)))
			return authorizedKeysMap, nil
		}
		authorizedKeysMap[string(publicKey.Marshal())] = true
		authorizedKeysBytes = rest

	}
	return authorizedKeysMap, nil
}

func (s *JupyterHubServer) GenConnConfig() *ssh.ClientConfig {
	clientConfig := &ssh.ClientConfig{User: s.connUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(s.connPasswd),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		BannerCallback: ssh.BannerDisplayStderr(),
	}

	return clientConfig
}
