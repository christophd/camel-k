//go:build integration
// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "integration"

/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package knative

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	messaging "knative.dev/eventing/pkg/apis/messaging/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

// Test that a Pipe can be changed and the changes are propagated to the Integration
func TestKameletChange(t *testing.T) {
	g := NewWithT(t)
	timerPipe := "timer-binding"

	knChannel := "test-kamelet-messages"
	knChannelConf := fmt.Sprintf("%s:InMemoryChannel:%s", messaging.SchemeGroupVersion.String(), knChannel)
	timerSource := "my-timer-source"
	g.Expect(CreateTimerKamelet(t, operatorID, ns, timerSource)()).To(Succeed())
	g.Expect(CreateKnativeChannel(t, ns, knChannel)()).To(Succeed())
	// Consumer route that will read from the KNative channel
	g.Expect(KamelRunWithID(t, operatorID, ns, "files/test-kamelet-display.groovy", "-w").Execute()).To(Succeed())
	g.Eventually(IntegrationPodPhase(t, ns, "test-kamelet-display")).Should(Equal(corev1.PodRunning))

	// Create the Pipe
	g.Expect(KamelBindWithID(t, operatorID, ns,
		timerSource,
		knChannelConf,
		"-p", "source.message=HelloKNative!",
		"--annotation", "trait.camel.apache.org/health.enabled=true",
		"--annotation", "trait.camel.apache.org/health.readiness-initial-delay=10",
		"--name", timerPipe,
	).Execute()).To(Succeed())
	g.Eventually(IntegrationPodPhase(t, ns, timerPipe)).Should(Equal(corev1.PodRunning))
	g.Eventually(IntegrationConditionStatus(t, ns, timerPipe, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
	// Consume the message
	g.Eventually(IntegrationLogs(t, ns, "test-kamelet-display"), TestTimeoutShort).Should(ContainSubstring("HelloKNative!"))

	g.Eventually(PipeCondition(t, ns, timerPipe, v1.PipeConditionReady), TestTimeoutMedium).Should(And(
		WithTransform(PipeConditionStatusExtract, Equal(corev1.ConditionTrue)),
		WithTransform(PipeConditionReason, Equal(v1.IntegrationConditionDeploymentReadyReason)),
		WithTransform(PipeConditionMessage, Equal(fmt.Sprintf("1/1 ready replicas"))),
	))

	// Update the Pipe
	g.Expect(KamelBindWithID(t, operatorID, ns,
		timerSource,
		knChannelConf,
		"-p", "source.message=message is Hi",
		"--annotation", "trait.camel.apache.org/health.enabled=true",
		"--annotation", "trait.camel.apache.org/health.readiness-initial-delay=10",
		"--name", timerPipe,
	).Execute()).To(Succeed())

	g.Eventually(IntegrationPodPhase(t, ns, timerPipe), TestTimeoutLong).Should(Equal(corev1.PodRunning))
	g.Eventually(IntegrationConditionStatus(t, ns, timerPipe, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
	g.Eventually(IntegrationLogs(t, ns, "test-kamelet-display"), TestTimeoutShort).Should(ContainSubstring("message is Hi"))

	g.Eventually(PipeCondition(t, ns, timerPipe, v1.PipeConditionReady), TestTimeoutMedium).
		Should(And(
			WithTransform(PipeConditionStatusExtract, Equal(corev1.ConditionTrue)),
			WithTransform(PipeConditionReason, Equal(v1.IntegrationConditionDeploymentReadyReason)),
			WithTransform(PipeConditionMessage, Equal("1/1 ready replicas")),
		))

	g.Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
}
