// Copyright 2025 Jelly Terra <jellyterra@symboltics.com>
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

//go:build !(android || linux)

package main

import (
	"context"
	"fmt"
)

func TriggerNotify(ctx context.Context, c chan<- struct{}) error {
	switch {
	case *triggerTime != 0:
		return TimerNotify(ctx, c)
	}
	return fmt.Errorf("no trigger specified")
}
