/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"net/url"
	"testing"
)

func TestGenerateCrateStorageURL(t *testing.T) {
	var value = `create storage {\"config\":{\"namespace\":\"/lindb-storage\",\"timeout\":10,\"dialTimeout\":10,\"leaseTTL\":10,\"endpoints\":[\"http://etcd:2379\"]}}`
	aa := url.Values{}
	aa.Add("sql", url.QueryEscape(value))
	t.Logf("%s", aa.Encode())
}
