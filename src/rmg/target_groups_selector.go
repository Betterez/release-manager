package main

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

type TargetGroupSelector struct {
	SelectedSourceGroups         []*elbv2.TargetGroup
	SelectedTargetGroups         []*elbv2.TargetGroup
	awsSession                   *session.Session
	allTargetGroupsForTheAccount *elbv2.DescribeTargetGroupsOutput
	environment                  string
	path                         string
	elbType                      string
}

type TargetGroupSearchResult int

const (
	TargetGroupNotFound TargetGroupSearchResult = iota
	TargetGroupFoundNonRelease
	TargetGroupFoundRelease
)

func (thisSelector *TargetGroupSelector) init(awsSession *session.Session, environment, path, elbType string) error {
	if awsSession == nil {
		return errors.New("Bad session object")
	}
	thisSelector.awsSession = awsSession
	thisSelector.SelectedSourceGroups = make([]*elbv2.TargetGroup, 0)
	thisSelector.SelectedTargetGroups = make([]*elbv2.TargetGroup, 0)
	thisSelector.path = path
	thisSelector.environment = environment
	thisSelector.elbType = elbType
	return nil
}

func (thisSelector *TargetGroupSelector) getallTargetGroupsForTheAccount() error {
	elbService := elbv2.New(thisSelector.awsSession)
	var err error
	thisSelector.allTargetGroupsForTheAccount, err = elbService.DescribeTargetGroups(&elbv2.DescribeTargetGroupsInput{})
	if err != nil {
		return err
	}
	return nil
}

func (thisSelector *TargetGroupSelector) checkTargetGroupsForMatch() error {
	thisSelector.getallTargetGroupsForTheAccount()
	elbService := elbv2.New(thisSelector.awsSession)
	for _, currentTargetGroup := range thisSelector.allTargetGroupsForTheAccount.TargetGroups {
		targetGroupTags, err := elbService.DescribeTags(&elbv2.DescribeTagsInput{
			ResourceArns: []*string{currentTargetGroup.TargetGroupArn},
		})
		if err != nil {
			return err
		}
		switch thisSelector.checkTargetGroupTagsForMatch(targetGroupTags) {
		case TargetGroupFoundRelease:
			thisSelector.SelectedSourceGroups = append(thisSelector.SelectedSourceGroups, currentTargetGroup)
			break
		case TargetGroupFoundNonRelease:
			thisSelector.SelectedTargetGroups = append(thisSelector.SelectedTargetGroups, currentTargetGroup)
			break
		default:
			break
		}
	}
	return nil
}

func (thisSelector *TargetGroupSelector) checkTargetGroupTagsForMatch(targetGroupTags *elbv2.DescribeTagsOutput) TargetGroupSearchResult {
	numberOfMatchingTags := 0
	isAReleaseTargetGroup := false
	for _, tagDescription := range targetGroupTags.TagDescriptions {
		for _, currentTagData := range tagDescription.Tags {
			switch *currentTagData.Key {
			case "Release":
				if *currentTagData.Value == "yes" {
					isAReleaseTargetGroup = true
					numberOfMatchingTags++
				}
				if *currentTagData.Value == "no" {
					isAReleaseTargetGroup = false
					numberOfMatchingTags++
				}
				break
			case "Environment":
				if *currentTagData.Value == thisSelector.environment {
					numberOfMatchingTags++
				}
				break
			case "Elb-Type":
				if *currentTagData.Value == thisSelector.elbType {
					numberOfMatchingTags++
				}
				break
			case "Path-Name":
				if *currentTagData.Value == thisSelector.path {
					numberOfMatchingTags++
				}
				break
			}
		}
	}
	if numberOfMatchingTags == 4 {
		if isAReleaseTargetGroup {
			return TargetGroupFoundRelease
		}
		return TargetGroupFoundNonRelease
	}
	return TargetGroupNotFound
}

func (thisSelector *TargetGroupSelector) GetTargetGroupsName() map[string][]string {
	result := make(map[string][]string, 0)
	for _, sourceGroup := range thisSelector.SelectedSourceGroups {
		result["source"] = append(result["source"], *sourceGroup.TargetGroupName)
	}
	for _, targetGroup := range thisSelector.SelectedTargetGroups {
		result["target"] = append(result["target"], *targetGroup.TargetGroupName)
	}
	return result
}
