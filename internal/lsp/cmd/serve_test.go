// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/tools/internal/jsonrpc2"
	"golang.org/x/tools/internal/tool"
)

func TestListenParsing(t *testing.T) {
	tests := []struct {
		input, wantNetwork, wantAddr string
	}{
		{"127.0.0.1:0", "tcp", "127.0.0.1:0"},
		{"unix;/tmp/sock", "unix", "/tmp/sock"},
		{"auto", "auto", ""},
		{"auto;foo", "auto", "foo"},
	}

	for _, test := range tests {
		gotNetwork, gotAddr := parseAddr(test.input)
		if gotNetwork != test.wantNetwork {
			t.Errorf("network = %q, want %q", gotNetwork, test.wantNetwork)
		}
		if gotAddr != test.wantAddr {
			t.Errorf("addr = %q, want %q", gotAddr, test.wantAddr)
		}
	}
}

// TestServeNoPortFlag checks that the serve command no longer declares the
// -port debugging flag, which listened on ":<port>" (all network interfaces).
func TestServeNoPortFlag(t *testing.T) {
	typ := reflect.TypeOf(Serve{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if name, ok := field.Tag.Lookup("flag"); ok && name == "port" {
			t.Errorf("Serve.%s declares the -port flag, which may bind all network interfaces", field.Name)
		}
	}
}

// TestServeListenRejectsImplicitAllInterfaces checks that a -listen address
// with no host, which implicitly binds all network interfaces (INADDR_ANY),
// is rejected instead of being served.
func TestServeListenRejectsImplicitAllInterfaces(t *testing.T) {
	for _, listen := range []string{":0", "tcp;:0", "tcp4;:0"} {
		// If the address were accepted, the server would listen until the
		// context expires; the timeout bounds the test in that case.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		s := &Serve{app: New("gopls-test", "", nil, nil)}
		err := tool.Run(ctx, s, []string{"-listen=" + listen})
		cancel()
		if err == nil || !strings.Contains(err.Error(), "implicitly binds all network interfaces") {
			t.Errorf("serve -listen=%s: got error %v, want implicit all-interfaces rejection", listen, err)
		}
	}
}

// TestServeListenExplicitHost checks that -listen addresses with an explicit
// host are still served.
func TestServeListenExplicitHost(t *testing.T) {
	for _, listen := range []string{"127.0.0.1:0", "tcp;127.0.0.1:0"} {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		s := &Serve{app: New("gopls-test", "", nil, nil)}
		err := tool.Run(ctx, s, []string{"-listen=" + listen, "-listen.timeout=10ms"})
		cancel()
		if err != jsonrpc2.ErrIdleTimeout {
			t.Errorf("serve -listen=%s: got error %v, want %v", listen, err, jsonrpc2.ErrIdleTimeout)
		}
	}
}
