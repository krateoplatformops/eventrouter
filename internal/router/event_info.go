package router

import (
	corev1 "k8s.io/api/core/v1"
)

type InvolvedObject struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	UID        string `json:"uid"`
}

type EventInfo struct {
	Type           string         `json:"type"`
	Reason         string         `json:"reason"`
	DeploymentId   string         `json:"deploymentId"`
	LastTimestamp  int64          `json:"lastTimestamp"`
	Message        string         `json:"message"`
	InvolvedObject InvolvedObject `json:"involvedObject"`
}

func NewEventInfo(deploymentID string, evt *corev1.Event) EventInfo {
	return EventInfo{
		Type:          evt.Type,
		Reason:        evt.Reason,
		DeploymentId:  deploymentID,
		LastTimestamp: evt.LastTimestamp.Time.Unix(),
		Message:       evt.Message,
		InvolvedObject: InvolvedObject{
			APIVersion: evt.InvolvedObject.APIVersion,
			Kind:       evt.InvolvedObject.Kind,
			Name:       evt.InvolvedObject.Name,
			UID:        string(evt.InvolvedObject.UID),
		},
	}
}
