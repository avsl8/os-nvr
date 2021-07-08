// Copyright 2020-2021 The OS-NVR Authors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; version 2.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package nvr

import (
	"context"
	"nvr/pkg/monitor"
	"nvr/pkg/storage"
	"nvr/pkg/web"
)

type envHook func(*storage.ConfigEnv)

type hookList struct {
	onEnvLoad          []envHook
	template           []web.Hook
	monitorStart       []monitor.StartHook
	monitorMainProcess []monitor.StartInputHook
	monitorSubProcess  []monitor.StartInputHook
}

var hooks = &hookList{}

// RegisterTplHook registers hook that's called when environment config is loaded.
func RegisterEnvHook(h envHook) {
	hooks.onEnvLoad = append(hooks.onEnvLoad, h)
}

// RegisterTplHook registers hook that's called on page render.
func RegisterTplHook(h web.Hook) {
	hooks.template = append(hooks.template, h)
}

// RegisterMonitorHook registers hook that's called when the monitor starts.
func RegisterMonitorStartHook(h monitor.StartHook) {
	hooks.monitorStart = append(hooks.monitorStart, h)
}

// RegisterMonitorHook registers hook that's called when the main monitor process starts.
func RegisterMonitorMainProcessHook(h monitor.StartInputHook) {
	hooks.monitorMainProcess = append(hooks.monitorMainProcess, h)
}

func RegisterMonitorSubProcessHook(h monitor.StartInputHook) {
	hooks.monitorSubProcess = append(hooks.monitorSubProcess, h)
}

func (h *hookList) env(env *storage.ConfigEnv) {
	for _, hook := range h.onEnvLoad {
		hook(env)
	}
}

func (h *hookList) tpl(pageFiles map[string]string) error {
	for _, hook := range h.template {
		if err := hook(pageFiles); err != nil {
			return err
		}
	}
	return nil
}

func (h *hookList) monitor() monitor.Hooks {
	startHook := func(ctx context.Context, m *monitor.Monitor) {
		for _, hook := range h.monitorStart {
			hook(ctx, m)
		}
	}
	startMainHook := func(ctx context.Context, m *monitor.Monitor, args *string) {
		for _, hook := range h.monitorMainProcess {
			hook(ctx, m, args)
		}
	}
	startSubHook := func(ctx context.Context, m *monitor.Monitor, args *string) {
		for _, hook := range h.monitorSubProcess {
			hook(ctx, m, args)
		}
	}

	return monitor.Hooks{
		Start:     startHook,
		StartMain: startMainHook,
		StartSub:  startSubHook,
	}
}
