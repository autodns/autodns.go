// Copyright 2025 Jelly Terra <jellyterra@proton.me>
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

//go:build android || linux

package main

import (
	"context"
	"syscall"
)

func Trigger(ctx context.Context, _ int, c chan<- struct{}) error {
	return NetlinkNotify(ctx, c)
}

func NetlinkNotify(ctx context.Context, c chan<- struct{}) error {
	fd, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_DGRAM, syscall.NETLINK_ROUTE)
	if err != nil {
		return err
	}
	defer syscall.Close(fd)

	err = syscall.Bind(fd, &syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
		Pid:    0,
		Groups: (1 << (syscall.RTNLGRP_LINK - 1)) | (1 << (syscall.RTNLGRP_IPV4_IFADDR - 1)) | (1 << (syscall.RTNLGRP_IPV6_IFADDR - 1)),
	})
	if err != nil {
		return err
	}

	for {
		packet := make([]byte, 2048)

		n, err := syscall.Read(fd, packet)
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		default:
		}

		messages, err := syscall.ParseNetlinkMessage(packet[:n])
		if err != nil {
			return err
		}

		for _, message := range messages {
			if message.Header.Type == syscall.RTM_NEWADDR || message.Header.Type == syscall.RTM_DELADDR || message.Header.Type == syscall.RTM_GETADDR {
				c <- struct{}{}
				break
			}
		}
	}
}
