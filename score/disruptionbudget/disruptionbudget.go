package disruptionbudget

import (
	"fmt"

	ks "github.com/zegl/kube-score/domain"
	"github.com/zegl/kube-score/score/checks"
	"github.com/zegl/kube-score/score/internal"
	"github.com/zegl/kube-score/scorecard"

	appsv1 "k8s.io/api/apps/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Register(allChecks *checks.Checks, budgets ks.PodDisruptionBudgets) {
	allChecks.RegisterStatefulSetCheck("StatefulSet has PodDisruptionBudget", `Makes sure that all StatefulSets are targeted by a PDB`, statefulSetHas(budgets.PodDisruptionBudgets()))
	allChecks.RegisterDeploymentCheck("Deployment has PodDisruptionBudget", `Makes sure that all Deployments are targeted by a PDB`, deploymentHas(budgets.PodDisruptionBudgets()))
}

func hasMatching(budgets []policyv1beta1.PodDisruptionBudget, namespace string, lables map[string]string) (bool, error) {
	for _, budget := range budgets {
		if budget.Namespace != namespace {
			continue
		}

		selector, err := metav1.LabelSelectorAsSelector(budget.Spec.Selector)
		if err != nil {
			return false, fmt.Errorf("failed to create selector: %v", err)
		}

		if selector.Matches(internal.MapLables(lables)) {
			return true, nil
		}
	}

	return false, nil
}

func statefulSetHas(budgets []policyv1beta1.PodDisruptionBudget) func(appsv1.StatefulSet) (scorecard.TestScore, error) {
	return func(statefulset appsv1.StatefulSet) (score scorecard.TestScore, err error) {
		if statefulset.Spec.Replicas != nil && *statefulset.Spec.Replicas < 2 {
			score.Grade = scorecard.GradeAllOK
			score.AddComment("", "Skipped", "Skipped because the statefulset has less than 2 replicas")
			return
		}

		match, matchErr := hasMatching(budgets, statefulset.Namespace, statefulset.Spec.Template.Labels)
		if matchErr != nil {
			err = matchErr
			return
		}

		if match {
			score.Grade = scorecard.GradeAllOK
		} else {
			score.Grade = scorecard.GradeCritical
			score.AddComment("", "No matching PodDisruptionBudget was found", "It's recommended to define a PodDisruptionBudget to avoid unexpected downtime during Kubernetes maintenance operations, such as when draining a node.")
		}

		return
	}
}

func deploymentHas(budgets []policyv1beta1.PodDisruptionBudget) func(appsv1.Deployment) (scorecard.TestScore, error) {
	return func(deployment appsv1.Deployment) (score scorecard.TestScore, err error) {
		if deployment.Spec.Replicas != nil && *deployment.Spec.Replicas < 2 {
			score.Grade = scorecard.GradeAllOK
			score.AddComment("", "Skipped", "Skipped because the deployment has less than 2 replicas")
			return
		}

		match, matchErr := hasMatching(budgets, deployment.Namespace, deployment.Spec.Template.Labels)
		if matchErr != nil {
			err = matchErr
			return
		}

		if match {
			score.Grade = scorecard.GradeAllOK
		} else {
			score.Grade = scorecard.GradeCritical
			score.AddComment("", "No matching PodDisruptionBudget was found", "It's recommended to define a PodDisruptionBudget to avoid unexpected downtime during Kubernetes maintenance operations, such as when draining a node.")
		}

		return
	}
}
