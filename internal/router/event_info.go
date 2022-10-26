package router

import (
	"time"

	corev1 "k8s.io/api/core/v1"
)

type InvolvedObject struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	UID        string `json:"uid"`
}

type Metadata struct {
	CreationTimestamp time.Time `json:"creationTimestamp"`
	Name              string    `json:"name"`
	Namespace         string    `json:"namespace"`
	UID               string    `json:"uid"`
}

type EventInfo struct {
	Type           string         `json:"type"`
	Reason         string         `json:"reason"`
	DeploymentId   string         `json:"deploymentId"`
	Time           int64          `json:"time"`
	Message        string         `json:"message"`
	Source         string         `json:"source"`
	InvolvedObject InvolvedObject `json:"involvedObject"`
	Metadata       Metadata       `json:"metadata"`
}

func NewEventInfo(deploymentID string, evt *corev1.Event) EventInfo {
	return EventInfo{
		Type:         evt.Type,
		Reason:       evt.Reason,
		DeploymentId: deploymentID,
		Time:         evt.LastTimestamp.Time.Unix(),
		Message:      evt.Message,
		Source:       evt.Source.Component,
		InvolvedObject: InvolvedObject{
			APIVersion: evt.InvolvedObject.APIVersion,
			Kind:       evt.InvolvedObject.Kind,
			Name:       evt.InvolvedObject.Name,
			UID:        string(evt.InvolvedObject.UID),
		},
		Metadata: Metadata{
			CreationTimestamp: evt.ObjectMeta.CreationTimestamp.Time,
			Name:              evt.ObjectMeta.Name,
			Namespace:         evt.ObjectMeta.Namespace,
			UID:               string(evt.ObjectMeta.UID),
		},
	}
}
