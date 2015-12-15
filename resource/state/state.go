// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state

import (
	"github.com/juju/loggo"
)

var logger = loggo.GetLogger("juju.resource.state")

// RawState defines the functionality needed from state.State for resources.
type RawState interface {
	// TODO(ericsnow) Add sub-interfaces here.
}

// State exposes the state functionality needed for resources.
type State struct {
	// TODO(ericsnow) Embed sub-structs here.
}

// NewState returns a new State for the given raw Juju state.
func NewState(raw RawState) *State {
	logger.Tracef("wrapping state for resources")
	return &State{
	// TODO(ericsnow) Add sub-structs here.
	}
}
