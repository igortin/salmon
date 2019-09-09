package salmon

import (
	"github.com/sirupsen/logrus"
	"sync"
)

type proxyLogger struct {
	proxyServer IoProxy
	log         *logrus.Logger
}
// Method that decorate logging servers
func (cmd *proxyLogger) Run(wg *sync.WaitGroup) error {
	cmd.log.Debug("Run server")
	err := cmd.proxyServer.Run(wg)
	if err != nil {
		cmd.log.Warn(err.Error())
		return err
	}
	return nil
}
// Method Not Used
func (cmd *proxyLogger) Shutdown() error {
	return nil
}
// Func to return instance proxyLogger
func NewProxyLogger(proxyServer IoProxy, log *logrus.Logger) IoProxy {
	return &proxyLogger{
		proxyServer,
		log,
	}
}