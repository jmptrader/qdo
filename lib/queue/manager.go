package queue

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/borgenk/qdo/third_party/github.com/syndtr/goleveldb/leveldb"

	"github.com/borgenk/qdo/lib/log"
)

type Manager struct {
	Conveyors map[string]*Conveyor
}

var (
	manager *Manager
	db      *leveldb.DB
	err     error
)

func StartManager() error {
	manager = &Manager{
		Conveyors: make(map[string]*Conveyor),
	}
	return manager.start()
}

func (man *Manager) start() error {
	dbFile := "/tmp/my.db"

	db, err = leveldb.OpenFile(dbFile, nil)
	if err != nil {
		log.Error(fmt.Sprintf("open database file %s failed", dbFile), err)
		return err
	}
	defer db.Close()

	iter := db.NewIterator(nil)
	for iter.Seek(nil); iter.Valid(); iter.Next() {
		db.Delete(iter.Key(), nil)
	}
	iter.Release()

	// TODO: Read conveyors from file here.
	storedConveyors := make(map[string]string)

	key := ""
	for _, v := range storedConveyors {
		if key == "" {
			key = v
		} else {
			man.Conveyors[key] = &Conveyor{}
			err := json.Unmarshal([]byte(v), man.Conveyors[key])
			if err != nil {
				log.Error("", err)
				return err
			}
			go man.Conveyors[key].Start()

			key = ""
		}
	}

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)
	<-exit

	return nil
}

func GetConveyor(conveyorID string) (*Conveyor, error) {
	if manager == nil {
		return nil, errors.New("Manager not initialized")
	}

	conv, ok := manager.Conveyors[conveyorID]
	if !ok {
		return nil, nil
	}

	return conv, nil
}

func GetAllConveyor() []*Conveyor {
	if manager == nil {
		return nil
	}

	resp := make([]*Conveyor, 0, len(manager.Conveyors))
	for _, v := range manager.Conveyors {
		resp = append(resp, v)
	}
	return resp
}

func AddConveyor(conveyorID string, config *Config) error {
	// TODO: Implement locking.
	manager.Conveyors[conveyorID] = NewConveyor(conveyorID, config)
	go manager.Conveyors[conveyorID].Start()

	_, err := json.Marshal(manager.Conveyors[conveyorID])
	// b
	if err != nil {
		log.Error("", err)
		return err
	}

	// TODO: Sync conveyor to file here.
	// conveyorID, b
	return nil
}

func RemoveConveyor(conveyorID string) error {
	if manager == nil {
		return errors.New("Manager not initialized")
	}

	conv, ok := manager.Conveyors[conveyorID]
	if !ok {
		return errors.New("Conveyor does not exist")
	}

	// TODO Sync conveyor to file here.

	go conv.Stop()

	return nil
}
