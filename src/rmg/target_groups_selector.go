package main

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

// TargetGroupSelector - filter target groups based on tags
type TargetGroupSelector struct {
	SelectedSourceGroups []*elbv2.TargetGroup
	SelectedTargetGroups []*elbv2.TargetGroup
	allTargetGroups      *elbv2.DescribeTargetGroupsOutput
	awsSession           *session.Session
	environment          string
	path                 string
	elbType              string
}

// TargetGroupSearchResult - results from target group search
type TargetGroupSearchResult int

const (
	// TargetGroupNotFound - target group not found
	TargetGroupNotFound TargetGroupSearchResult = iota
	// TargetGroupFoundNonRelease -source (not release) tg
	TargetGroupFoundNonRelease
	// TargetGroupFoundRelease - release tg
	TargetGroupFoundRelease
)

func (tgs *TargetGroupSelector) init(sess *session.Session, environment, path, elbType string) {
	tgs.awsSession = sess
	tgs.SelectedSourceGroups = make([]*elbv2.TargetGroup, 0)
	tgs.SelectedTargetGroups = make([]*elbv2.TargetGroup, 0)
	tgs.path = path
	tgs.environment = environment
	tgs.elbType = elbType
}

func (tgs *TargetGroupSelector) getAllTargetGroups() error {
	elbService := elbv2.New(tgs.awsSession)
	var err error
	tgs.allTargetGroups, err = elbService.DescribeTargetGroups(&elbv2.DescribeTargetGroupsInput{})
	if err != nil {
		return err
	}
	return nil
}

func (tgs *TargetGroupSelector) checkTargetGroupsForMatch() error {
	elbService := elbv2.New(tgs.awsSession)
	for _, currentTargetGroupt := range tgs.allTargetGroups.TargetGroups {
		targetGroupTags, err := elbService.DescribeTags(&elbv2.DescribeTagsInput{
			ResourceArns: []*string{currentTargetGroupt.TargetGroupArn},
		})
		if err != nil {
			return err
		}
		switch tgs.checkTargetGroupTagsForMatch(targetGroupTags) {
		case TargetGroupFoundRelease:
			tgs.SelectedTargetGroups = append(tgs.SelectedTargetGroups, currentTargetGroupt)
			break
		case TargetGroupFoundNonRelease:
			tgs.SelectedSourceGroups = append(tgs.SelectedSourceGroups, currentTargetGroupt)
			break
		default:
			break
		}
	}
	return nil
}

func (tgs *TargetGroupSelector) checkTargetGroupTagsForMatch(targetGroupTags *elbv2.DescribeTagsOutput) TargetGroupSearchResult {
	tagNameValues := []string{"Environment", "Elb-Type", "Path-Name", "Release"}
	tagValues := []string{tgs.environment, tgs.elbType, tgs.path, "yes"}
	numberOfMatchingTags := 0
	isAReleaseTargetGroup := false
	for _, tagDescription := range targetGroupTags.TagDescriptions {
		if numberOfMatchingTags == 4 {
			break
		}
		for _, tagMeta := range tagDescription.Tags {
			for i, tagName := range tagNameValues {
				if *tagMeta.Key == "Release" {
					numberOfMatchingTags++
					if *tagMeta.Value == "yes" {
						isAReleaseTargetGroup = true
					} else {
						isAReleaseTargetGroup = false
					}
				} else if *tagMeta.Key == tagName && *tagMeta.Value == tagValues[i] {
					numberOfMatchingTags++
				}
				if numberOfMatchingTags == len(tagNameValues) {
					if !isAReleaseTargetGroup {
						return TargetGroupFoundRelease
					}
					return TargetGroupFoundNonRelease
				}
			}
		}
	}
	return TargetGroupNotFound
}
