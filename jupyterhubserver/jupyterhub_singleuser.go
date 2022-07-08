package jupyterhubserver

import (
	"fmt"

	"golang.org/x/crypto/ssh"
)

type SingleUser struct {
	username          string
	password          string
	authorizedKeysMap map[string]bool
	podName           string
	podIP             string
	client            *ssh.Client
}

func NewSingleUser(username string, password string, authorizedKeysMap map[string]bool, podIP string, client *ssh.Client) *SingleUser {
	podName := fmt.Sprintf("jupyter-%s", username)
	return &SingleUser{username: username,
		password:          password,
		authorizedKeysMap: authorizedKeysMap,
		podName:           podName,
		podIP:             podIP,
		client:            client}
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

func (u *SingleUser) CheckAuthorizedKey(key string) bool {
	return u.authorizedKeysMap[key]
}
