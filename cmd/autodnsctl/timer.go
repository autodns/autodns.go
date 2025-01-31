// Copyright 2025 Jelly Terra <jellyterra@symboltics.com>
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"time"
)

func TimerNotify(ctx context.Context, c chan<- struct{}) error {
	for {
		select {
		case <-time.After(time.Duration(*triggerTime) * time.Second):
			c <- struct{}{}
		case <-ctx.Done():
			return nil
		}
	}
}
