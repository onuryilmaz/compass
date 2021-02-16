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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/kyma-incubator/compass/components/director/pkg/graphql"
	web_hook "github.com/kyma-incubator/compass/components/director/pkg/webhook"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"io/ioutil"
	"net/http"
	"time"
)

// Client defines a general purpose Webhook executor
type Client interface {
	Do(ctx context.Context, request *Request) (*web_hook.Response, error) // TODO: Move Response and other templates to a better package - maybe even in the Controller project itself
}

type DefaultClient struct {
}

func (d DefaultClient) Do(ctx context.Context, request *Request) (*web_hook.Response, error) {
	webhook := request.Webhook

	var method string
	url := webhook.URL
	if webhook.URLTemplate != nil {
		resultURL, err := web_hook.ParseURLTemplate(webhook.URLTemplate, request.Data)
		if err != nil {
			return nil, errors.Wrap(err, "unable to parse webhook URL")
		}
		url = resultURL.Path
		method = *resultURL.Method
	}

	if url == nil {
		return nil, errors.New("missing webhook url")
	}

	body, err := web_hook.ParseInputTemplate(webhook.InputTemplate, request.Data)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse webhook input body")
	}

	headers, err := web_hook.ParseHeadersTemplate(webhook.HeaderTemplate, request.Data)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse webhook headers")
	}

	if webhook.CorrelationIDKey != nil && request.CorrelationID != "" {
		headers.Add(*webhook.CorrelationIDKey, request.CorrelationID)
	}

	req, err := http.NewRequestWithContext(ctx, method, *url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header = headers

	client := http.DefaultClient
	if webhook.Auth != nil {
		basicCreds, isBasicAuth := webhook.Auth.Credential.(graphql.BasicCredentialData)
		if isBasicAuth {
			req.SetBasicAuth(basicCreds.Username, basicCreds.Password)
		}

		oauthCreds, isOAuth := webhook.Auth.Credential.(graphql.OAuthCredentialData)
		if isOAuth {
			client = oauthClient(ctx, *client, oauthCreds)
		}
	}

	resp, err := http.DefaultClient.Do(req) // TODO: Build custom client, do not rely on default one
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var respBody map[string]interface{}
	if err := json.Unmarshal(bytes, &respBody); err != nil {
		return nil, err
	}

	responseData := web_hook.ResponseData{
		Body:    respBody,
		Headers: resp.Header,
	}

	response, err := web_hook.ParseOutputTemplate(webhook.InputTemplate, webhook.OutputTemplate, web_hook.Mode(*webhook.Mode), responseData)
	if err != nil {
		return nil, err
	}

	var recErr *ReconcileError
	if *response.SuccessStatusCode != resp.StatusCode {
		recErr = &ReconcileError{Description: fmt.Sprintf("response success status code was not met - expected %q, got %q", *response.SuccessStatusCode, resp.StatusCode)}
	}

	if response.Error != nil && *response.Error != "" {
		recErr = &ReconcileError{Description: fmt.Sprintf("received error while requesting external system: %s", *response.Error)}
	}

	if recErr != nil && !isWebhookTimeoutReached(request.OperationCreationTime, time.Duration(*webhook.Timeout)) {
		recErr.Requeue = true
		recErr.RequeueAfter = request.RetryInterval
		return nil, recErr
	}

	return response, recErr
}

func isWebhookTimeoutReached(creationTime time.Time, webhookTimeout time.Duration) bool {
	operationEndTime := creationTime.Add(webhookTimeout)
	return time.Now().After(operationEndTime)
}

func oauthClient(ctx context.Context, client http.Client, oauthCreds graphql.OAuthCredentialData) *http.Client {
	conf := &clientcredentials.Config{
		ClientID:     oauthCreds.ClientID,
		ClientSecret: oauthCreds.ClientSecret,
		TokenURL:     oauthCreds.URL,
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, client)
	securedClient := conf.Client(ctx)
	securedClient.Timeout = client.Timeout
	return securedClient
}
