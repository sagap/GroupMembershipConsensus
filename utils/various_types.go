package utils

type ClusterMessage struct {
	Source  string `json:"source"`
	Dest    string `json:"dest"`
	Message string   `json:"message"`
}