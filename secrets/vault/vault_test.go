// Copyright 2019 The Go Cloud Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limtations under the License.

package vault

import (
	"context"
	"testing"

	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/builtin/logical/transit"
	vhttp "github.com/hashicorp/vault/http"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/vault"
	"gocloud.dev/secrets/driver"
	"gocloud.dev/secrets/drivertest"
)

const (
	keyID      = "test-secrets"
	apiAddress = "http://127.0.0.1:0"
)

type harness struct {
	client *api.Client
	close  func()
}

func (h *harness) MakeDriver(ctx context.Context) (driver.Keeper, error) {
	return newKeeper(keyID, h.client), nil
}

func (h *harness) Close() {
	h.close()
}

func newKeeper(keyID string, client *api.Client) driver.Keeper {
	return &keeper{
		keyID:  keyID,
		client: client,
	}
}

func newHarness(ctx context.Context, t *testing.T) (drivertest.Harness, error) {
	// Start a new test server.
	c, cleanup := testVaultServer(t)
	// Enable the Transit Secrets Engine to use Vault as an Encryption as a Service.
	c.Logical().Write("sys/mounts/transit", map[string]interface{}{
		"type": "transit",
	})

	return &harness{
		client: c,
		close:  cleanup,
	}, nil
}

func testVaultServer(t *testing.T) (*api.Client, func()) {
	coreCfg := &vault.CoreConfig{
		DisableMlock: true,
		DisableCache: true,
		// Enable the testing transit backend.
		LogicalBackends: map[string]logical.Factory{
			"transit": transit.Factory,
		},
	}
	cluster := vault.NewTestCluster(t, coreCfg, &vault.TestClusterOptions{
		HandlerFunc: vhttp.Handler,
	})
	cluster.Start()
	tc := cluster.Cores[0]
	vault.TestWaitActive(t, tc.Core)

	tc.Client.SetToken(cluster.RootToken)
	return tc.Client, cluster.Cleanup
}

func TestConformance(t *testing.T) {
	drivertest.RunConformanceTests(t, newHarness)
}
