// Copyright 2025 Jelly Terra <jellyterra@proton.me>
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/autodns/autodns.go/core"
)

func HandleWrap(handler func(w http.ResponseWriter, r *http.Request) (int, error, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code, err, iErr := handler(w, r)
		switch {
		case iErr != nil:
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println(iErr)
		case err != nil:
			if code != 0 {
				w.WriteHeader(code)
			} else {
				w.WriteHeader(http.StatusBadRequest)
			}
			_, _ = w.Write([]byte(MarshalJSON(&struct {
				Error string `json:"error"`
			}{
				Error: err.Error(),
			})))
		default:
			_, _ = w.Write([]byte("{}"))
		}
	}
}

type ReqDo struct {
	Role  string `json:"role"`
	Token string `json:"token"`

	Operations []*core.Operation `json:"operations"`
}

func Serve(ctx context.Context, c *core.Context, addr string, route string) error {
	mux := http.NewServeMux()

	mux.HandleFunc(path.Join(route, "/v1/do"), HandleWrap(func(w http.ResponseWriter, r *http.Request) (int, error, error) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			return 0, err, nil
		}

		req, err := UnmarshalJSON(b, &ReqDo{})
		if err != nil {
			return 0, err, nil
		}

		roleDef, err := core.Query(c, &core.RoleDef{}, "role", req.Role)
		switch {
		case err == nil:
		case os.IsNotExist(err):
			return http.StatusUnauthorized, fmt.Errorf("authorization failed"), nil
		default:
			return 0, nil, err
		}

		key, ok := roleDef.Keys[req.Token]
		if !ok || key.Expire != 0 && key.Expire < time.Now().Unix() {
			return http.StatusUnauthorized, fmt.Errorf("authorization failed"), nil
		}

		err = core.ExecuteAll(c, roleDef, req.Operations, func(err error, op *core.Operation) {
			switch op.Op {
			case core.OP_UPDATE:
				fmt.Printf("Role [%s] updates [%s] => [%s] with TTL [%d]", req.Role, op.CanonicalName, op.Value, op.TTL)
			case core.OP_DELETE:
				fmt.Printf("Role [%s] deletes [%s]", req.Role, op.CanonicalName)
			}
			if err != nil {
				fmt.Println(", failed:", err)
			} else {
				fmt.Print("\n")
			}
		})
		if err != nil {
			return 0, nil, err
		}

		return 0, nil, nil
	}))

	s := http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		fmt.Println("Shutting down")
		_ = s.Shutdown(ctx)
	}()

	err := s.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
