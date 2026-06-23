// Copyright The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package remote

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	remoteapi "github.com/prometheus/client_golang/exp/api/remote"
	"github.com/prometheus/common/promslog"
	"github.com/stretchr/testify/require"

	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/prompb"
)

func TestRemoteWriteHandlerLabelLimits(t *testing.T) {
	payload, _, _, err := buildWriteRequest(nil, []prompb.TimeSeries{{
		Labels:  []prompb.Label{{Name: "this_is_a_very_long_label_name", Value: "valid"}},
		Samples: []prompb.Sample{{Value: 1, Timestamp: 1000}},
	}}, nil, nil, nil, nil, "snappy")
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader(payload))
	require.NoError(t, err)

	appendable := &mockAppendable{}
	cfg := func() config.Config {
		return config.Config{
			GlobalConfig: config.GlobalConfig{
				LabelNameLengthLimit:  10,
				LabelValueLengthLimit: 15,
			},
		}
	}
	handler := NewWriteHandler(promslog.NewNopLogger(), nil, appendable, []remoteapi.WriteMessageType{remoteapi.WriteV1MessageType}, false, false, false, cfg)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	resp := recorder.Result()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	require.Contains(t, string(body), "label length limit exceeded")
}
