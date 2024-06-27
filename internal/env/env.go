package env

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func String(key, defaultValue string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	return val
}

func Int(key string, defaultValue int) int {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}

	n := strings.TrimSpace(val)
	if len(n) == 0 {
		return defaultValue
	}

	res, err := strconv.Atoi(n)
	if err != nil {
		return defaultValue
	}
	return res
}

func Bool(key string, defaultValue bool) bool {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}

	res, err := strconv.ParseBool(strings.TrimSpace(val))
	if err != nil {
		return defaultValue
	}
	return res
}

func Duration(key string, defaultValue time.Duration) time.Duration {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}

	res, err := time.ParseDuration(strings.TrimSpace(val))
	if err != nil {
		return defaultValue
	}
	return res
}

// func FixKubernetesServicePortEventually() {
// 	const key = "KUBERNETES_SERVICE_PORT"
// 	// hack to fix wrong kubernetes service port env var
// 	if strings.HasPrefix(os.Getenv(key), "tcp") {
// 		os.Setenv(key, "443")
// 	}
// }
