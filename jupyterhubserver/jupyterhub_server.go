package jupyterhubserver

import (
	"fmt"
	"net/http"
	"regexp"

	log "github.com/lylelaii/golang_utils/logger/v1"
	requestes "github.com/lylelaii/golang_utils/requestes/v1"
	"golang.org/x/crypto/ssh"
)

const MODULENAME = "jupyterhubserver"

type JupyterHubServerConfig struct {
	Url        string `mapstructure:"url"`
	AdminToken string `mapstructure:"admin_token"`
	ConnUser   string `mapstructure:"conn_user"`
	ConnPasswd string `mapstructure:"conn_passwd"`
	SshPort    string `mapstructure:"ssh_port"`
	VerifyTLS  bool   `mapstructure:"verify_tls"`
}

type JupyterHubServer struct {
	url             string
	adminToken      string
	connUser        string
	connPasswd      string
	sshPort         string
	requestesClient *requestes.RequestsClient
	logger          log.Logger
}

type SingleUser struct {
	username string
	password string
	podName  string
	podIP    string
	client   *ssh.Client
}

func NewJupyterHubServer(c JupyterHubServerConfig, logger log.Logger) *JupyterHubServer {
	requestesClinet, _ := requestes.New(requestes.RequestsConfig{VerifyTLS: c.VerifyTLS})
	return &JupyterHubServer{url: c.Url,
		adminToken:      c.AdminToken,
		connUser:        c.ConnUser,
		connPasswd:      c.ConnPasswd,
		sshPort:         c.SshPort,
		requestesClient: requestesClinet,
		logger:          logger}
}

func NewSingleUser(username string, password string, podIP string, client *ssh.Client) *SingleUser {
	podName := fmt.Sprintf("jupyter-%s", username)
	return &SingleUser{username: username, password: password, podName: podName, podIP: podIP, client: client}
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
	headers["Accept"] = "application/jupyterhub-pagination+json"

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

	return true, &userInfo

}

func (s *JupyterHubServer) queryUserRoute(username string) *UserRoute {
	var headers map[string]string = make(map[string]string)
	headers["Authorization"] = fmt.Sprintf("token %s", s.adminToken)
	headers["Accept"] = "application/jupyterhub-pagination+json"

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

	return &r

}

func (s *JupyterHubServer) GetPodIP(username string) string {
	userRoute := s.queryUserRoute(username)

	target := userRoute.Target
	ipReg := `((2(5[0-5]|[0-4]\d))|[0-1]?\d{1,2})(\.((2(5[0-5]|[0-4]\d))|[0-1]?\d{1,2})){3}`
	reg, _ := regexp.Compile(ipReg)
	podIP := reg.Find([]byte(target))

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

func (u *SingleUser) GetClient() *ssh.Client {
	return u.client
}

func (u *SingleUser) GetPodName() string {
	return u.podName
}

func (u *SingleUser) GetPodIP() string {
	return u.podIP
}

func (u *SingleUser) UpdateClient(client *ssh.Client) {
	u.client = client
}
