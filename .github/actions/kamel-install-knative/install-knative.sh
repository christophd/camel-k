#!/bin/bash

# ---------------------------------------------------------------------------
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# ---------------------------------------------------------------------------

####
#
# Install the knative setup
#
####

set -e

# Prerequisites
sudo wget https://github.com/mikefarah/yq/releases/download/v4.9.6/yq_linux_amd64 -O /usr/bin/yq && sudo chmod +x /usr/bin/yq

export SERVING_VERSION=knative-v1.1.0
export EVENTING_VERSION=knative-v1.1.0
export KOURIER_VERSION=knative-v1.1.0

# Serving
kubectl apply --filename https://github.com/knative/serving/releases/download/${SERVING_VERSION}/serving-crds.yaml
curl -L -s https://github.com/knative/serving/releases/download/${SERVING_VERSION}/serving-core.yaml | head -n -1 | yq e 'del(.spec.template.spec.containers[].resources)' - | kubectl apply -f -

# Kourier
kubectl apply --filename https://github.com/knative-sandbox/net-kourier/releases/download/${KOURIER_VERSION}/kourier.yaml
kubectl patch configmap/config-network \
  --namespace knative-serving \
  --type merge \
  --patch '{"data":{"ingress.class":"kourier.ingress.networking.knative.dev"}}'

# Eventing
kubectl apply --filename https://github.com/knative/eventing/releases/download/${EVENTING_VERSION}/eventing-crds.yaml
curl -L -s https://github.com/knative/eventing/releases/download/${EVENTING_VERSION}/eventing-core.yaml | head -n -1 | yq e 'del(.spec.template.spec.containers[].resources)' - | kubectl apply -f -

# Eventing channels
curl -L -s https://github.com/knative/eventing/releases/download/${EVENTING_VERSION}/in-memory-channel.yaml | head -n -1 | yq e 'del(.spec.template.spec.containers[].resources)' - | kubectl apply -f -

# Eventing broker
curl -L -s https://github.com/knative/eventing/releases/download/${EVENTING_VERSION}/mt-channel-broker.yaml | head -n -1 | yq e 'del(.spec.template.spec.containers[].resources)' - | kubectl apply -f -

# Eventing sugar controller for injection
kubectl apply -f https://github.com/knative/eventing/releases/download/${EVENTING_VERSION}/eventing-sugar-controller.yaml

# Wait for installation completed
echo "Waiting for all pods to be ready in kourier-system"
kubectl wait --for=condition=Ready pod --all -n kourier-system --timeout=60s
echo "Waiting for all pods to be ready in knative-serving"
kubectl wait --for=condition=Ready pod --all -n knative-serving --timeout=60s
echo "Waiting for all pods to be ready in knative-eventing"
kubectl wait --for=condition=Ready pod --all -n knative-eventing --timeout=60s
