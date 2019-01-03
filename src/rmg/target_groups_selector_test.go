package main

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

func createTagsOutput() *elbv2.DescribeTagsOutput {
	return &elbv2.DescribeTagsOutput{
		TagDescriptions: []*elbv2.TagDescription{{
			Tags: []*elbv2.Tag{
				{
					Key:   aws.String("Environment"),
					Value: aws.String("production"),
				},
				{
					Key:   aws.String("Elb-Type"),
					Value: aws.String("api"),
				},
				{
					Key:   aws.String("Path-Name"),
					Value: aws.String("reports"),
				},
				{
					Key:   aws.String("Release"),
					Value: aws.String("yes"),
				},
			},
		}},
	}
}

func createSelector() *TargetGroupSelector {
	return &TargetGroupSelector{
		elbType:     "api",
		environment: "staging",
		path:        "notificaitons",
	}
}

func TestNegativeReleaseTag(t *testing.T) {
	testValue := createTagsOutput()
	selector := createSelector()
	testValue.TagDescriptions[0].Tags[3].Value = aws.String("no")
	result := selector.checkTargetGroupTagsForMatch(testValue)
	if result == TargetGroupFoundRelease {
		t.Error("Not a release group come back as release")
	}
}

func TestPositiveReleaseTag(t *testing.T) {
	testValue := createTagsOutput()
	selector := createSelector()
	testValue.TagDescriptions[0].Tags[2].Value = aws.String("notificaitons")
	testValue.TagDescriptions[0].Tags[0].Value = aws.String("staging")
	result := selector.checkTargetGroupTagsForMatch(testValue)
	if result != TargetGroupFoundRelease {
		t.Error("release group not found")
	}
}

func TestWrongEnvironment(t *testing.T) {
	testValue := createTagsOutput()
	selector := createSelector()
	testValue.TagDescriptions[0].Tags[2].Value = aws.String("notificaitons")
	result := selector.checkTargetGroupTagsForMatch(testValue)
	if result != TargetGroupNotFound {
		t.Errorf("Found tg in the wrong environment: %s should not be found in %s",
			selector.environment,
			*testValue.TagDescriptions[0].Tags[0].Value,
		)
	}
}
func TestWrongPath(t *testing.T) {
	testValue := createTagsOutput()
	selector := createSelector()
	testValue.TagDescriptions[0].Tags[0].Value = aws.String("staging")
	result := selector.checkTargetGroupTagsForMatch(testValue)
	if result != TargetGroupNotFound {
		t.Errorf("Found tg in the wrong path: %s should not be found in %s",
			selector.path,
			*testValue.TagDescriptions[0].Tags[2].Value,
		)
	}
}

func TestWrongELBType(t *testing.T) {
	testValue := createTagsOutput()
	selector := createSelector()
	selector.elbType = "app"
	testValue.TagDescriptions[0].Tags[0].Value = aws.String("staging")
	result := selector.checkTargetGroupTagsForMatch(testValue)
	if result != TargetGroupNotFound {
		t.Errorf("Found tg in the wrong elb type: %s should not be found in %s",
			selector.elbType,
			*testValue.TagDescriptions[0].Tags[1].Value,
		)
	}
}
