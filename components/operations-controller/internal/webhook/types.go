/*
 * Copyright 2020 The Compass Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package webhook

import (
	"github.com/kyma-incubator/compass/components/director/pkg/graphql"
	web_hook "github.com/kyma-incubator/compass/components/director/pkg/webhook"
)

// Request represents a webhook request to be executed
type Request struct {
	Webhook       graphql.Webhook
	Data          web_hook.RequestData
	CorrelationID string
	PollURL       *string
}

func NewRequest(webhook graphql.Webhook, requestData web_hook.RequestData, correlationID string) *Request {
	return &Request{
		Webhook:       webhook,
		Data:          requestData,
		CorrelationID: correlationID,
	}
}
