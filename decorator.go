package salmon

import (
	"github.com/sirupsen/logrus"
	"sync"
)


// ProxyLogger is decorator for IoProxy interface with log support.
type ProxyLogger struct {
	ProxyServer IoProxy
	Log         *logrus.Logger
}

// Run proxy server with log.
func (cmd *ProxyLogger) Run(wg *sync.WaitGroup) error {
	cmd.Log.Debug("Run server")
	err := cmd.ProxyServer.Run(wg)
	if err != nil {
		cmd.Log.Warn(err.Error())
		return err
	}
	return nil
}
// Shutdown is not implemented.
func (cmd *ProxyLogger) Shutdown() error {
	return nil
}

// NewProxyLogger return proxyLogger object.
func NewProxyLogger(proxyServer IoProxy, log *logrus.Logger) IoProxy {
	return &ProxyLogger{
		proxyServer,
		log,
	}}