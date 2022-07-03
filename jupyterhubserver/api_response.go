package jupyterhubserver

import (
	"encoding/json"
	"time"
)

type UserInfo struct {
	Admin        bool          `json:"admin,omitempty"`
	AuthState    interface{}   `json:"auth_state,omitempty"`
	Created      time.Time     `json:"created,omitempty"`
	Groups       []interface{} `json:"groups,omitempty"`
	Kind         string        `json:"kind,omitempty"`
	LastActivity time.Time     `json:"last_activity,omitempty"`
	Name         string        `json:"name,omitempty"`
	Pending      interface{}   `json:"pending,omitempty"`
	Server       string        `json:"server,omitempty"`
	Servers      Servers       `json:"servers,omitempty"`
}

type Servers struct {
	ServerDetail `json:",omitempty"`
}

func (s *Servers) UnmarshalJSON(b []byte) error {
	var d map[string]ServerDetail
	err := json.Unmarshal(b, &d)
	if err == nil {
		s.ServerDetail = d[""]
	}
	return err
}

type ServerDetail struct {
	LastActivity time.Time   `json:"last_activity,omitempty"`
	Name         string      `json:"name,omitempty"`
	Pending      interface{} `json:"pending,omitempty"`
	ProgressURL  string      `json:"progress_url,omitempty"`
	Ready        bool        `json:"ready,omitempty"`
	Started      time.Time   `json:"started,omitempty"`
	State        State       `json:"state,omitempty"`
	URL          string      `json:"url,omitempty"`
	UserOptions  UserOptions `json:"user_options,omitempty"`
}

type State struct {
	PodName string `json:"pod_name,omitempty"`
}
type UserOptions struct {
	Profile string `json:"profile,omitempty"`
}

type UserRoutes map[string]UserRoute

type UserRoute struct {
	Routespec string `json:"routespec"`
	Target    string `json:"target"`
	Data      struct {
		User         string    `json:"user"`
		ServerName   string    `json:"server_name"`
		LastActivity time.Time `json:"last_activity"`
	} `json:"data"`
}
