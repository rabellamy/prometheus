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

package promql

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/util/teststorage"
)

func TestLabelLimits(t *testing.T) {
	storage := teststorage.New(t)
	defer storage.Close()

	app := storage.Appender(context.Background())
	_, err := app.Append(0, labels.FromStrings("__name__", "up", "instance", "localhost"), time.Now().UnixMilli(), 1.0)
	require.NoError(t, err)
	require.NoError(t, app.Commit())

	opts := EngineOpts{
		Logger:             nil,
		Reg:                nil,
		MaxSamples:         1000,
		Timeout:            10 * time.Second,
		ActiveQueryTracker: nil,
		LookbackDelta:      5 * time.Minute,
	}

	engine := NewEngine(opts)
	// Set strict limits for testing.
	engine.SetLabelLimits(10, 15)

	testCases := []struct {
		name          string
		query         string
		expectedError string
	}{
		{
			name:          "label_replace exceed name limit",
			query:         `label_replace(up, "this_is_a_very_long_label_name", "1", "", "")`,
			expectedError: "label name length limit exceeded: 30 > 10",
		},
		{
			name:          "label_replace exceed value limit",
			query:         `label_replace(up, "validname", "this_is_a_very_long_label_value", "", "")`,
			expectedError: "label value length limit exceeded: 31 > 15",
		},
		{
			name:          "label_join exceed name limit",
			query:         `label_join(up, "this_is_a_very_long_label_name", ",", "instance")`,
			expectedError: "label name length limit exceeded: 30 > 10",
		},
		{
			name:          "label_join exceed value limit",
			query:         `label_join(up, "validname", "this_is_a_very_long_separator_that_causes_value_to_exceed_limit", "instance", "instance")`,
			expectedError: "label value length limit exceeded",
		},
		{
			name:          "valid label_replace",
			query:         `label_replace(up, "shortname", "shortval", "", "")`,
			expectedError: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			qry, err := engine.NewInstantQuery(context.Background(), storage, nil, tc.query, time.Now())
			require.NoError(t, err)

			res := qry.Exec(context.Background())
			if tc.expectedError != "" {
				require.Error(t, res.Err)
				require.Contains(t, res.Err.Error(), tc.expectedError)
			} else {
				require.NoError(t, res.Err)
			}
			qry.Close()
		})
	}
}
