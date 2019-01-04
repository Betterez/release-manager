package main

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

// InstancesSwitcher - deploy instances from the source group to the target one
type InstancesSwitcher struct {
	awsSession                     *session.Session
	SelectedSourceGroups           []*elbv2.TargetGroup
	SelectedTargetGroups           []*elbv2.TargetGroup
	sourceInstancesDescriptions    []*elbv2.TargetDescription
	targetInstancesMapDescriptions map[string][]*elbv2.TargetDescription
}

// Init - initialize
func (is *InstancesSwitcher) Init(sess *session.Session, sourceTargetGroups, targetTargetGroups []*elbv2.TargetGroup) error {
	if sess == nil || sourceTargetGroups == nil || targetTargetGroups == nil {
		return errors.New("parameters nil error")
	}
	if len(sourceTargetGroups) == 0 || len(targetTargetGroups) == 0 {
		return errors.New("empty source or target group")
	}
	if len(sourceTargetGroups) > 1 {
		return errors.New("source groups cna only contain 1 group")
	}
	is.awsSession = sess
	is.SelectedSourceGroups = sourceTargetGroups
	is.SelectedTargetGroups = targetTargetGroups
	is.targetInstancesMapDescriptions = make(map[string][]*elbv2.TargetDescription, 0)
	return nil
}

// SwitchInstances - switch instances from the source group to the rtarget groups
func (is *InstancesSwitcher) SwitchInstances() error {
	if len(is.sourceInstancesDescriptions) == 0 {
		return errors.New("no source instances")
	}
	return nil
}

func (is *InstancesSwitcher) getInstancesInGroups() error {
	is.sourceInstancesDescriptions = make([]*elbv2.TargetDescription, 0)
	if err := getInstancesDescriptionForGroups(is.SelectedSourceGroups, &is.sourceInstancesDescriptions, is.awsSession); err != nil {
		return err
	}
	var targetGroupDescription []*elbv2.TargetDescription
	for _, targetGroup := range is.SelectedTargetGroups {
		fmt.Printf("checking %s\r\n", *targetGroup.TargetGroupName)
		if err := getInstancesDescriptionForGroups([]*elbv2.TargetGroup{targetGroup},
			&targetGroupDescription,
			is.awsSession); err != nil {
			return err
		}
		is.targetInstancesMapDescriptions[*targetGroup.TargetGroupName] = targetGroupDescription
		//fmt.Println("targetGroupDescription", targetGroupDescription, "targetGroupDescription")
	}

	return nil
}

func getInstancesDescriptionForGroups(targetGroups []*elbv2.TargetGroup,
	instancesDescriptions *[]*elbv2.TargetDescription,
	awsSession *session.Session) error {
	elbService := elbv2.New(awsSession)

	for _, currentTargetGroup := range targetGroups {
		targetDescription, err := elbService.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
			TargetGroupArn: currentTargetGroup.TargetGroupArn,
		})
		if err != nil {
			return err
		}
		for _, desc := range targetDescription.TargetHealthDescriptions {
			if *desc.TargetHealth.State == "healthy" {
				*instancesDescriptions = append(*instancesDescriptions, desc.Target)
			}
		}
	}
	return nil
}
