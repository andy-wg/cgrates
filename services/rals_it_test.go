// +build integration

/*
Real-time Online/Offline Charging System (OCS) for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/
package services

import (
	"path"
	"testing"
	"time"

	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/servmanager"
	"github.com/cgrates/cgrates/utils"
	"github.com/cgrates/rpcclient"
)

func TestRalsReload(t *testing.T) {
	cfg, err := config.NewDefaultCGRConfig()
	if err != nil {
		t.Fatal(err)
	}
	utils.Newlogger(utils.MetaSysLog, cfg.GeneralCfg().NodeID)
	utils.Logger.SetLogLevel(7)
	filterSChan := make(chan *engine.FilterS, 1)
	filterSChan <- nil
	engineShutdown := make(chan bool, 1)
	chS := engine.NewCacheS(cfg, nil)
	close(chS.GetPrecacheChannel(utils.CacheThresholdProfiles))
	close(chS.GetPrecacheChannel(utils.CacheThresholds))
	close(chS.GetPrecacheChannel(utils.CacheThresholdFilterIndexes))

	close(chS.GetPrecacheChannel(utils.CacheDestinations))
	close(chS.GetPrecacheChannel(utils.CacheReverseDestinations))
	close(chS.GetPrecacheChannel(utils.CacheRatingPlans))
	close(chS.GetPrecacheChannel(utils.CacheRatingProfiles))
	close(chS.GetPrecacheChannel(utils.CacheActions))
	close(chS.GetPrecacheChannel(utils.CacheActionPlans))
	close(chS.GetPrecacheChannel(utils.CacheAccountActionPlans))
	close(chS.GetPrecacheChannel(utils.CacheActionTriggers))
	close(chS.GetPrecacheChannel(utils.CacheSharedGroups))
	close(chS.GetPrecacheChannel(utils.CacheTimings))

	cfg.ChargerSCfg().Enabled = true
	cfg.ThresholdSCfg().Enabled = true
	cacheSChan := make(chan rpcclient.RpcClientConnection, 1)
	cacheSChan <- chS
	server := utils.NewServer()
	srvMngr := servmanager.NewServiceManager(cfg /*dm*/, nil,
		/*cdrStorage*/ nil,
		/*loadStorage*/ nil, filterSChan,
		server, nil, engineShutdown)
	srvMngr.SetCacheS(chS)
	ralS := NewRalService(srvMngr)
	srvMngr.AddService(ralS, NewChargerService(), &CacheService{connChan: cacheSChan}, NewSchedulerService(), NewThresholdService())
	if err = srvMngr.StartServices(); err != nil {
		t.Error(err)
	}
	if ralS.IsRunning() {
		t.Errorf("Expected service to be down")
	}
	var reply string
	if err := cfg.V1ReloadConfig(&config.ConfigReloadWithArgDispatcher{
		Path:    path.Join("/usr", "share", "cgrates", "conf", "samples", "tutmongo"),
		Section: config.RALS_JSN,
	}, &reply); err != nil {
		t.Error(err)
	} else if reply != utils.OK {
		t.Errorf("Expecting OK ,received %s", reply)
	}
	time.Sleep(10 * time.Millisecond) //need to switch to gorutine
	if !ralS.IsRunning() {
		t.Errorf("Expected service to be running")
	}

	if apiv1, has := srvMngr.GetService(utils.ApierV1); !has {
		t.Error("Expected to find service")
	} else if !apiv1.IsRunning() {
		t.Errorf("Expected service to be running")
	}

	if apiv2, has := srvMngr.GetService(utils.ApierV2); !has {
		t.Error("Expected to find service")
	} else if !apiv2.IsRunning() {
		t.Errorf("Expected service to be running")
	}

	if resp, has := srvMngr.GetService(utils.ResponderS); !has {
		t.Error("Expected to find service")
	} else if !resp.IsRunning() {
		t.Errorf("Expected service to be running")
	}

	cfg.RalsCfg().Enabled = false
	cfg.GetReloadChan(config.RALS_JSN) <- struct{}{}
	time.Sleep(10 * time.Millisecond)
	if ralS.IsRunning() {
		t.Errorf("Expected service to be down")
	}

	if apiv1, has := srvMngr.GetService(utils.ApierV1); !has {
		t.Error("Expected to find service")
	} else if apiv1.IsRunning() {
		t.Errorf("Expected service to be down")
	}

	if apiv2, has := srvMngr.GetService(utils.ApierV2); !has {
		t.Error("Expected to find service")
	} else if apiv2.IsRunning() {
		t.Errorf("Expected service to be down")
	}

	if resp, has := srvMngr.GetService(utils.ResponderS); !has {
		t.Error("Expected to find service")
	} else if resp.IsRunning() {
		t.Errorf("Expected service to be down")
	}
	engineShutdown <- true
}