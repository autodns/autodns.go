// Copyright 2025 Jelly Terra <jellyterra@proton.me>
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

//go:build !(android || linux)

package main

import (
	"context"
	"time"
)

func Trigger(ctx context.Context, triggerTime int, c chan<- struct{}) error {
	return TimerNotify(ctx, triggerTime, c)
}

func TimerNotify(ctx context.Context, triggerTime int, c chan<- struct{}) error {
	for {
		select {
		case <-time.After(time.Duration(triggerTime) * time.Second):
			c <- struct{}{}
		case <-ctx.Done():
			return nil
		}
	}
}
