package main

import (
	"testing"
)

func TestPromql(t *testing.T) {
	expect := "histogram_quantile(0.9000, sum by (le, method, path) (rate(demo_api_request_duration_seconds_bucket[5m])))"
	q1 := Func{
		Fun: "histogram_quantile",
	}.WithParameters(
		Scalar(0.9),
		AggregationOp{Operator: "sum"}.
			WithByClause("le", "method", "path").
			SetOperand(
				Func{Fun: "rate"}.WithParameters(TSSelector{Name: "demo_api_request_duration_seconds_bucket"}.WithDuration("5m")),
			),
	)
	if actual := q1.String(); actual != expect {
		t.Fatalf("expect: %s, actual: %s", expect, actual)
	}

	expect = "sum by (job, mode) (rate(node_cpu_seconds_total[1m])) / on(job) group_left sum by (job) (rate(node_cpu_seconds_total[1m]))"
	q2 := BinaryOp{Operator: "/"}.
		WithMatcher(VectorMatcher{Keyword: "on", Labels: []string{"job"}}.WithGroupLeft()).
		WithOperands(
			AggregationOp{Operator: "sum"}.
				WithByClause("job", "mode").
				SetOperand(Func{Fun: "rate"}.WithParameters(TSSelector{Name: "node_cpu_seconds_total"}.WithDuration("1m"))),
			AggregationOp{Operator: "sum"}.
				WithByClause("job").
				SetOperand(Func{Fun: "rate"}.WithParameters(TSSelector{Name: "node_cpu_seconds_total"}.WithDuration("1m"))),
		)
	if actual := q2.String(); actual != expect {
		t.Fatalf("expect: %s, actual: %s", expect, actual)
	}
}