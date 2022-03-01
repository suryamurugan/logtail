/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tail

import (
	"fmt"
	"sync"

	"github.com/vogo/logger"
	"github.com/vogo/logtail/internal/conf"
	"github.com/vogo/logtail/internal/trans"
	"github.com/vogo/logtail/internal/util"
)

// DefaultTailer the default tailer.
// nolint:gochecknoglobals // ignore this
var DefaultTailer *Tailer

// Tailer the logtail tailer.
type Tailer struct {
	lock      sync.Mutex
	Config    *conf.Config
	Servers   map[string]*Server
	Transfers map[string]trans.Transfer
}

// NewTailer new logtail tailer.
func NewTailer(config *conf.Config) (*Tailer, error) {
	if err := conf.InitialCheckConfig(config); err != nil {
		return nil, err
	}

	tailer := &Tailer{
		lock:      sync.Mutex{},
		Config:    config,
		Servers:   make(map[string]*Server, util.DefaultMapSize),
		Transfers: make(map[string]trans.Transfer, util.DefaultMapSize),
	}

	return tailer, nil
}

func (r *Tailer) Start() error {
	conf.ConfigLogLevel(r.Config.LogLevel)

	if err := r.startTransfers(); err != nil {
		return err
	}

	for _, serverConfig := range r.Config.Servers {
		_, err := r.AddServer(serverConfig)
		if err != nil {
			logger.Errorf("add server %s error: %v", serverConfig.Name, err)
		}
	}

	return nil
}

func (r *Tailer) startTransfers() error {
	for _, c := range r.Config.Transfers {
		if _, err := r.StartTransfer(c); err != nil {
			return err
		}
	}

	return nil
}

func (r *Tailer) AddTransfer(c *conf.TransferConfig) error {
	if _, err := r.StartTransfer(c); err != nil {
		return err
	}

	r.Config.Transfers[c.Name] = c
	r.Config.SaveToFile()

	return nil
}

// nolint:ireturn //ignore this.
func (r *Tailer) StartTransfer(transferConfig *conf.TransferConfig) (trans.Transfer, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	runTransfer := BuildTransfer(transferConfig)

	if err := runTransfer.Start(); err != nil {
		logger.Infof("transfer [%s]%s Start error: %v", transferConfig.Type, runTransfer.Name(), err)

		return nil, err
	}

	logger.Infof("transfer [%s]%s started", transferConfig.Type, runTransfer.Name())

	existTransfer, exist := r.Transfers[transferConfig.Name]

	// save or replace transfer
	r.Transfers[transferConfig.Name] = runTransfer

	if exist {
		for _, server := range r.Servers {
			for _, router := range server.Routers {
				router.Lock.Lock()
				for i := range router.Transfers {
					if router.Transfers[i].Name() == runTransfer.Name() {
						// replace transfer
						router.Transfers[i] = runTransfer
					}
				}
				router.Lock.Unlock()
			}
		}

		// stop exists transfer
		_ = existTransfer.Stop()
	}

	return runTransfer, nil
}

func (r *Tailer) StopTransfer(name string) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if existTransfer, exist := r.Transfers[name]; exist {
		if r.isTransferUsing(name) {
			return fmt.Errorf("%w: %s", conf.ErrTransferUsing, name)
		}

		err := existTransfer.Stop()
		if err != nil {
			logger.Warnf("stop transfer error: %v", err)
		}

		delete(r.Transfers, name)

		delete(r.Config.Transfers, name)
		r.Config.SaveToFile()
	}

	return nil
}

func (r *Tailer) isTransferUsing(name string) bool {
	for _, server := range r.Servers {
		for _, router := range server.Routers {
			for i := range router.Transfers {
				if router.Transfers[i].Name() == name {
					return true
				}
			}
		}
	}

	return false
}

// Stop the runner.
func (r *Tailer) Stop() {
	r.lock.Lock()
	defer r.lock.Unlock()

	for _, s := range r.Servers {
		if err := s.Stop(); err != nil {
			logger.Errorf("server %s close error: %+v", s.ID, err)
		}
	}

	for _, t := range r.Transfers {
		if err := t.Stop(); err != nil {
			logger.Errorf("transfer %s close error: %+v", t.Name(), err)
		}
	}
}

func (r *Tailer) AddRouter(config *conf.RouterConfig) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if err := conf.CheckRouterConfig(r.Config, config); err != nil {
		return err
	}

	var err error

	if _, ok := r.Config.Routers[config.Name]; ok {
		for _, server := range r.Servers {
			for _, router := range server.Routers {
				if router.Name == config.Name {
					if err = server.AddRouter(config); err != nil {
						logger.Errorf("add router error: %v", err)
					}
				}
			}
		}
	}

	r.Config.Routers[config.Name] = config
	r.Config.SaveToFile()

	return err
}

func (r *Tailer) DeleteRouter(name string) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if _, exist := r.Config.Routers[name]; exist {
		if r.isRouterUsing(name) {
			return fmt.Errorf("%w: %s", conf.ErrRouterUsing, name)
		}

		delete(r.Config.Routers, name)
		r.Config.SaveToFile()
	}

	return nil
}

func (r *Tailer) isRouterUsing(name string) bool {
	for _, server := range r.Servers {
		for _, router := range server.Routers {
			if router.Name == name {
				return true
			}
		}
	}

	return false
}

func (r *Tailer) AddServer(serverConfig *conf.ServerConfig) (*Server, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if err := conf.CheckServerConfig(r.Config, serverConfig); err != nil {
		return nil, err
	}

	server := NewServer(serverConfig)

	format := serverConfig.Format
	if format == nil {
		format = r.Config.DefaultFormat
	}

	server.Format = format
	server.Tailer = r

	if existsServer, ok := r.Servers[server.ID]; ok {
		_ = existsServer.Stop()

		delete(r.Servers, server.ID)
	}

	r.Servers[server.ID] = server

	server.Initial(r.Config, serverConfig)

	server.Start()

	r.Config.Servers[serverConfig.Name] = serverConfig
	r.Config.SaveToFile()

	return server, nil
}

func (r *Tailer) DeleteServer(name string) error {
	s, exist := r.Servers[name]

	if exist {
		if err := s.Stop(); err != nil {
			return err
		}

		delete(r.Servers, name)

		delete(r.Config.Servers, name)
		r.Config.SaveToFile()
	}

	return nil
}