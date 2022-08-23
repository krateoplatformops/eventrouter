package router

import (
	"time"

	corev1 "k8s.io/api/core/v1"
)

type Notification struct {
	Level         string `json:"level"`
	Time          int64  `json:"time"`
	Message       string `json:"message"`
	Source        string `json:"source"`
	Reason        string `json:"reason"`
	TransactionId string `json:"deploymentId"`
}

func NewNotification(deploymentID string, evt *corev1.Event) Notification {
	level := "info"
	if evt.Type != corev1.EventTypeNormal {
		level = "error"
	}
	return Notification{
		Level:         level, //evt.Type,
		Time:          time.Now().Unix(),
		Source:        evt.InvolvedObject.Name,
		Reason:        evt.Reason,
		Message:       evt.Message,
		TransactionId: deploymentID,
	}
}
