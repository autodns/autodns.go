// Copyright 2025 Jelly Terra <jellyterra@symboltics.com>
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"fmt"
	"github.com/autodns/autodns.go/core"
	"io"
	"net/http"
	"path"
)

func HandleWrap(handler func(w http.ResponseWriter, r *http.Request) (int, error, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code, err, i_err := handler(w, r)
		switch {
		case i_err != nil:
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println(i_err)
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

func Serve(ctx context.Context, globalCache *core.GlobalCache, addr string, route string, db *core.Database) error {
	mux := http.NewServeMux()

	mux.HandleFunc(path.Join(route, "/v1/do"), HandleWrap(func(w http.ResponseWriter, r *http.Request) (int, error, error) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			return 0, err, nil
		}

		req, err := UnmarshalJSON(b, &ReqDo{})

		ok, err := db.MatchRoleToken(r.Context(), req.Role, req.Token)
		if err != nil {
			return 0, nil, err
		}
		if !ok {
			return http.StatusUnauthorized, fmt.Errorf("invalid token"), nil
		}

		err, i_err := core.ValidateAll(ctx, globalCache, db, req.Role, req.Operations)
		if i_err != nil {
			return 0, nil, i_err
		}
		if err != nil {
			return 0, err, nil
		}

		go func() {
			registries, err := core.BuildRegistries(ctx, db, req.Operations)
			if err != nil {
				fmt.Println("Building registries failed:", err)
				return
			}

			core.ExecuteAll(req.Operations, registries, func(err error, op *core.Operation) {
				switch op.Op {
				case core.OP_UPDATE:
					fmt.Printf("Role [%s] updates [%s] => [%s] with TTL [%d]", op.Role, op.CanonicalName, op.Value, op.TTL)
				case core.OP_DELETE:
					fmt.Printf("Role [%s] deletes [%s]", op.Role, op.CanonicalName)
				}
				if err != nil {
					fmt.Println(", failed:", err)
				} else {
					fmt.Print("\n")
				}
			})
		}()

		return 0, nil, nil
	}))

	s := http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		fmt.Println("Shutting down")
		_ = s.Shutdown(ctx)
	}()

	return s.ListenAndServe()
}
