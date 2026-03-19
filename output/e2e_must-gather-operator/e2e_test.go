// DO NOT REMOVE TAGS BELOW. IF ANY NEW TEST FILES ARE CREATED UNDER /test/e2e, PLEASE ADD THESE TAGS TO THEM IN ORDER TO BE EXCLUDED FROM UNIT TESTS.
//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mustgatherv1alpha1 "github.com/openshift/must-gather-operator/api/v1alpha1"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Diff-suggested: Test new metrics field in GatherSpec (EP-1906)
var _ = Describe("MustGather Metrics Flag", func() {
	const (
		timeout  = time.Minute * 5
		interval = time.Second * 10
	)

	var (
		testNamespace = "default"
		ctx           = context.Background()
	)

	Context("When creating a MustGather with metrics flag", func() {
		// Diff-suggested: Verify metrics field is accepted and persisted
		It("Should accept metrics=true in gatherSpec", func() {
			mgName := "test-metrics-basic"

			mustGather := &mustgatherv1alpha1.MustGather{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mgName,
					Namespace: testNamespace,
				},
				Spec: mustgatherv1alpha1.MustGatherSpec{
					GatherSpec: &mustgatherv1alpha1.GatherSpec{
						Metrics: true,
					},
				},
			}

			By("Creating MustGather with metrics=true")
			Expect(adminClient.Create(ctx, mustGather)).To(Succeed())

			By("Verifying the metrics field is persisted")
			createdMG := &mustgatherv1alpha1.MustGather{}
			Eventually(func() bool {
				err := adminClient.Get(ctx, types.NamespacedName{
					Name:      mgName,
					Namespace: testNamespace,
				}, createdMG)
				if err != nil {
					return false
				}
				return createdMG.Spec.GatherSpec != nil && createdMG.Spec.GatherSpec.Metrics
			}, timeout, interval).Should(BeTrue())

			By("Cleaning up")
			Expect(adminClient.Delete(ctx, mustGather)).To(Succeed())
		})

		// Diff-suggested: Verify backward compatibility - metrics is optional
		It("Should create MustGather without metrics field (backward compatibility)", func() {
			mgName := "test-no-metrics"

			mustGather := &mustgatherv1alpha1.MustGather{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mgName,
					Namespace: testNamespace,
				},
				Spec: mustgatherv1alpha1.MustGatherSpec{
					GatherSpec: &mustgatherv1alpha1.GatherSpec{
						Audit: false,
						// metrics field omitted
					},
				},
			}

			By("Creating MustGather without metrics field")
			Expect(adminClient.Create(ctx, mustGather)).To(Succeed())

			By("Verifying MustGather is created successfully")
			createdMG := &mustgatherv1alpha1.MustGather{}
			Eventually(func() error {
				return adminClient.Get(ctx, types.NamespacedName{
					Name:      mgName,
					Namespace: testNamespace,
				}, createdMG)
			}, timeout, interval).Should(Succeed())

			By("Cleaning up")
			Expect(adminClient.Delete(ctx, mustGather)).To(Succeed())
		})

		// Diff-suggested: Verify both audit and metrics can be used together
		It("Should accept both audit and metrics flags together", func() {
			mgName := "test-audit-and-metrics"

			mustGather := &mustgatherv1alpha1.MustGather{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mgName,
					Namespace: testNamespace,
				},
				Spec: mustgatherv1alpha1.MustGatherSpec{
					GatherSpec: &mustgatherv1alpha1.GatherSpec{
						Audit:   true,
						Metrics: true,
					},
				},
			}

			By("Creating MustGather with both audit and metrics flags")
			Expect(adminClient.Create(ctx, mustGather)).To(Succeed())

			By("Verifying both fields are persisted")
			createdMG := &mustgatherv1alpha1.MustGather{}
			Eventually(func() bool {
				err := adminClient.Get(ctx, types.NamespacedName{
					Name:      mgName,
					Namespace: testNamespace,
				}, createdMG)
				if err != nil {
					return false
				}
				return createdMG.Spec.GatherSpec != nil &&
					createdMG.Spec.GatherSpec.Audit &&
					createdMG.Spec.GatherSpec.Metrics
			}, timeout, interval).Should(BeTrue())

			By("Cleaning up")
			Expect(adminClient.Delete(ctx, mustGather)).To(Succeed())
		})
	})

	Context("When the controller processes MustGather with metrics flag", func() {
		// Diff-suggested: Verify controller sets GATHER_METRICS env var when metrics=true
		It("Should set GATHER_METRICS environment variable when metrics=true", func() {
			mgName := "test-metrics-env-var"

			mustGather := &mustgatherv1alpha1.MustGather{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mgName,
					Namespace: testNamespace,
				},
				Spec: mustgatherv1alpha1.MustGatherSpec{
					GatherSpec: &mustgatherv1alpha1.GatherSpec{
						Metrics: true,
					},
				},
			}

			By("Creating MustGather with metrics=true")
			Expect(adminClient.Create(ctx, mustGather)).To(Succeed())

			By("Waiting for the Job to be created")
			job := &batchv1.Job{}
			Eventually(func() error {
				return adminClient.Get(ctx, types.NamespacedName{
					Name:      mgName,
					Namespace: testNamespace,
				}, job)
			}, timeout, interval).Should(Succeed())

			By("Verifying GATHER_METRICS environment variable is set")
			var gatherContainer *corev1.Container
			for i := range job.Spec.Template.Spec.Containers {
				if job.Spec.Template.Spec.Containers[i].Name == gatherContainerName {
					gatherContainer = &job.Spec.Template.Spec.Containers[i]
					break
				}
			}
			Expect(gatherContainer).NotTo(BeNil(), "gather container should exist")

			// Check for GATHER_METRICS env var
			var metricsEnvVar *corev1.EnvVar
			for i := range gatherContainer.Env {
				if gatherContainer.Env[i].Name == "GATHER_METRICS" {
					metricsEnvVar = &gatherContainer.Env[i]
					break
				}
			}
			Expect(metricsEnvVar).NotTo(BeNil(), "GATHER_METRICS environment variable should be set")
			Expect(metricsEnvVar.Value).To(Equal("true"), "GATHER_METRICS should be 'true'")

			By("Cleaning up")
			Expect(adminClient.Delete(ctx, mustGather)).To(Succeed())
		})

		// Diff-suggested: Verify controller does NOT set GATHER_METRICS when metrics=false or omitted
		It("Should NOT set GATHER_METRICS environment variable when metrics is not specified", func() {
			mgName := "test-no-metrics-env-var"

			mustGather := &mustgatherv1alpha1.MustGather{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mgName,
					Namespace: testNamespace,
				},
				Spec: mustgatherv1alpha1.MustGatherSpec{},
			}

			By("Creating MustGather without metrics field")
			Expect(adminClient.Create(ctx, mustGather)).To(Succeed())

			By("Waiting for the Job to be created")
			job := &batchv1.Job{}
			Eventually(func() error {
				return adminClient.Get(ctx, types.NamespacedName{
					Name:      mgName,
					Namespace: testNamespace,
				}, job)
			}, timeout, interval).Should(Succeed())

			By("Verifying GATHER_METRICS environment variable is NOT set")
			var gatherContainer *corev1.Container
			for i := range job.Spec.Template.Spec.Containers {
				if job.Spec.Template.Spec.Containers[i].Name == gatherContainerName {
					gatherContainer = &job.Spec.Template.Spec.Containers[i]
					break
				}
			}
			Expect(gatherContainer).NotTo(BeNil(), "gather container should exist")

			// Verify GATHER_METRICS is NOT in env vars
			for _, envVar := range gatherContainer.Env {
				Expect(envVar.Name).NotTo(Equal("GATHER_METRICS"),
					"GATHER_METRICS should not be set when metrics field is not specified")
			}

			By("Cleaning up")
			Expect(adminClient.Delete(ctx, mustGather)).To(Succeed())
		})

		// Diff-suggested: Verify audit and metrics flags work independently
		It("Should set audit command and metrics env var independently", func() {
			// Test 1: audit only
			mgAuditOnly := &mustgatherv1alpha1.MustGather{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-audit-only",
					Namespace: testNamespace,
				},
				Spec: mustgatherv1alpha1.MustGatherSpec{
					GatherSpec: &mustgatherv1alpha1.GatherSpec{
						Audit: true,
					},
				},
			}

			By("Creating MustGather with audit=true only")
			Expect(adminClient.Create(ctx, mgAuditOnly)).To(Succeed())

			By("Verifying audit command is used but GATHER_METRICS is not set")
			jobAudit := &batchv1.Job{}
			Eventually(func() error {
				return adminClient.Get(ctx, types.NamespacedName{
					Name:      "test-audit-only",
					Namespace: testNamespace,
				}, jobAudit)
			}, timeout, interval).Should(Succeed())

			// Find gather container
			var gatherContainerAudit *corev1.Container
			for i := range jobAudit.Spec.Template.Spec.Containers {
				if jobAudit.Spec.Template.Spec.Containers[i].Name == gatherContainerName {
					gatherContainerAudit = &jobAudit.Spec.Template.Spec.Containers[i]
					break
				}
			}
			Expect(gatherContainerAudit).NotTo(BeNil())

			// Verify GATHER_METRICS is NOT set
			for _, envVar := range gatherContainerAudit.Env {
				Expect(envVar.Name).NotTo(Equal("GATHER_METRICS"))
			}

			By("Cleaning up audit-only test")
			Expect(adminClient.Delete(ctx, mgAuditOnly)).To(Succeed())

			// Test 2: metrics only
			mgMetricsOnly := &mustgatherv1alpha1.MustGather{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-metrics-only",
					Namespace: testNamespace,
				},
				Spec: mustgatherv1alpha1.MustGatherSpec{
					GatherSpec: &mustgatherv1alpha1.GatherSpec{
						Metrics: true,
					},
				},
			}

			By("Creating MustGather with metrics=true only")
			Expect(adminClient.Create(ctx, mgMetricsOnly)).To(Succeed())

			By("Verifying GATHER_METRICS is set")
			jobMetrics := &batchv1.Job{}
			Eventually(func() error {
				return adminClient.Get(ctx, types.NamespacedName{
					Name:      "test-metrics-only",
					Namespace: testNamespace,
				}, jobMetrics)
			}, timeout, interval).Should(Succeed())

			// Find gather container
			var gatherContainerMetrics *corev1.Container
			for i := range jobMetrics.Spec.Template.Spec.Containers {
				if jobMetrics.Spec.Template.Spec.Containers[i].Name == gatherContainerName {
					gatherContainerMetrics = &jobMetrics.Spec.Template.Spec.Containers[i]
					break
				}
			}
			Expect(gatherContainerMetrics).NotTo(BeNil())

			// Verify GATHER_METRICS IS set
			var metricsEnvFound bool
			for _, envVar := range gatherContainerMetrics.Env {
				if envVar.Name == "GATHER_METRICS" && envVar.Value == "true" {
					metricsEnvFound = true
					break
				}
			}
			Expect(metricsEnvFound).To(BeTrue(), "GATHER_METRICS should be set when metrics=true")

			By("Cleaning up metrics-only test")
			Expect(adminClient.Delete(ctx, mgMetricsOnly)).To(Succeed())
		})
	})

	Context("When MustGather job completes", func() {
		// Diff-suggested: Integration test - verify complete workflow with metrics
		It("Should complete successfully with metrics flag enabled", func() {
			mgName := "test-metrics-completion"

			mustGather := &mustgatherv1alpha1.MustGather{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mgName,
					Namespace: testNamespace,
				},
				Spec: mustgatherv1alpha1.MustGatherSpec{
					GatherSpec: &mustgatherv1alpha1.GatherSpec{
						Metrics: true,
					},
				},
			}

			By("Creating MustGather with metrics=true")
			Expect(adminClient.Create(ctx, mustGather)).To(Succeed())

			By("Waiting for the Job to be created")
			job := &batchv1.Job{}
			Eventually(func() error {
				return adminClient.Get(ctx, types.NamespacedName{
					Name:      mgName,
					Namespace: testNamespace,
				}, job)
			}, timeout, interval).Should(Succeed())

			By("Waiting for MustGather to complete")
			// Note: Depending on cluster resources, this may take several minutes
			// The job runs actual must-gather collection
			Eventually(func() bool {
				mg := &mustgatherv1alpha1.MustGather{}
				err := adminClient.Get(ctx, types.NamespacedName{
					Name:      mgName,
					Namespace: testNamespace,
				}, mg)
				if err != nil {
					return false
				}
				return mg.Status.Completed
			}, time.Minute*10, interval).Should(BeTrue())

			By("Verifying MustGather status is Completed")
			completedMG := &mustgatherv1alpha1.MustGather{}
			Expect(adminClient.Get(ctx, types.NamespacedName{
				Name:      mgName,
				Namespace: testNamespace,
			}, completedMG)).To(Succeed())
			Expect(completedMG.Status.Status).To(Equal("Completed"))

			By("Cleaning up")
			Expect(adminClient.Delete(ctx, mustGather)).To(Succeed())
		})
	})
})
