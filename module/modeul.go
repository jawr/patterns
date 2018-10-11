package module

import (
	"sync"
)

type Module interface {}

type manager struct {
	sync.RWMutex
	data map[string]Module
}

func (m *manager) Get(k string) Module {
	m.RLock()
	defer m.RUnlock()
	mod, ok := m.data[k]
	if !ok {
		panic("module: unable to find module '" + k + "'")
	}
	return mod
}

func (m *manager) Set(k string, mod Module) {
	m.Lock()
	defer m.Unlock()
	_, ok := m.data[k]
	if ok {
		panic("module: tried to store the same named module '" + k + "'")
	}
	m.data[k] = mod
}

var singleton *Manager
var once sync.Once

func getManager() *Manager {
	once.Do(func() {
		singleton = &Manager{
			data: make(map[string]Module, 0),
		}
	})

	return singleton
}

func GetModule(k string) Module {
	return getManager().Get(k)
}

func RegisterModule(k string, m Module) {
	return getManager().Set(k, m)
}
