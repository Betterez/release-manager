package main

import (
	"errors"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

// InstancesSwitcher - deploy instances from the source group to the target one
type InstancesSwitcher struct {
	awsSession                  *session.Session
	SelectedSourceTargetGroups  []*elbv2.TargetGroup
	SelectedTargetTargetGroups  []*elbv2.TargetGroup
	sourceInstancesDescriptions []*elbv2.TargetDescription
	// TODO: do we need this? ththisSwitcher might be removed
	targetInstancesMapDescriptions map[string][]*elbv2.TargetDescription
}

// Init - initialize
func (thisSwitcher *InstancesSwitcher) Init(sess *session.Session, sourceTargetGroups, targetTargetGroups []*elbv2.TargetGroup) error {
	if sess == nil || sourceTargetGroups == nil || targetTargetGroups == nil {
		return errors.New("parameters nil error")
	}
	if len(sourceTargetGroups) == 0 || len(targetTargetGroups) == 0 {
		return errors.New("empty source or target group")
	}
	if len(sourceTargetGroups) > 1 {
		return errors.New("source groups cna only contain 1 group")
	}
	thisSwitcher.awsSession = sess
	thisSwitcher.SelectedSourceTargetGroups = sourceTargetGroups
	thisSwitcher.SelectedTargetTargetGroups = targetTargetGroups
	thisSwitcher.targetInstancesMapDescriptions = make(map[string][]*elbv2.TargetDescription, 0)
	return nil
}

// SwitchInstances - switch instances from the source group to the rtarget groups
func (thisSwitcher *InstancesSwitcher) SwitchInstances() error {
	thisSwitcher.getInstancesInGroupsIfNeeded()
	if len(thisSwitcher.sourceInstancesDescriptions) == 0 {
		return errors.New("no source instances")
	}
	for _, targetGroup := range thisSwitcher.SelectedTargetTargetGroups {
		thisSwitcher.registerInstancesWithTargetGroup(targetGroup)
		thisSwitcher.removeOldInstancesFromTargetGroup(targetGroup)
	}
	thisSwitcher.cleanSourceTargetGroup()
	return nil
}

func (thisSwitcher *InstancesSwitcher) registerInstancesWithTargetGroup(targetTargetGroup *elbv2.TargetGroup) error {
	elbService := elbv2.New(thisSwitcher.awsSession)
	elbService.RegisterTargets(&elbv2.RegisterTargetsInput{
		TargetGroupArn: targetTargetGroup.TargetGroupArn,
		Targets:        thisSwitcher.sourceInstancesDescriptions,
	})
	healthyInstances := 0
	for {
		time.Sleep(5 * time.Second)
		instancesHealth, err := elbService.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
			TargetGroupArn: targetTargetGroup.TargetGroupArn,
			Targets:        thisSwitcher.sourceInstancesDescriptions,
		})
		if err != nil {
			return err
		}
		for _, instanceHealthDescription := range instancesHealth.TargetHealthDescriptions {
			if *instanceHealthDescription.TargetHealth.State == "healthy" {
				healthyInstances++
			}
		}
		if healthyInstances == len(thisSwitcher.sourceInstancesDescriptions) {
			break
		}
		healthyInstances = 0
	}
	return nil
}

func (thisSwitcher *InstancesSwitcher) removeOldInstancesFromTargetGroup(targetTargetGroup *elbv2.TargetGroup) error {
	elbService := elbv2.New(thisSwitcher.awsSession)
	var targetInstancesDescription, instancesToBeRemoved []*elbv2.TargetDescription
	getInstancesDescriptionForGroups([]*elbv2.TargetGroup{targetTargetGroup}, &targetInstancesDescription, thisSwitcher.awsSession)
	sourceInstancesMap := mapInstancesByID(thisSwitcher.sourceInstancesDescriptions)
	for _, instanceDescription := range targetInstancesDescription {
		if sourceInstancesMap[*instanceDescription.Id] > 0 {
			continue
		}
		instancesToBeRemoved = append(instancesToBeRemoved, instanceDescription)
	}
	if len(instancesToBeRemoved) > 0 {
		elbService.DeregisterTargets(&elbv2.DeregisterTargetsInput{
			TargetGroupArn: targetTargetGroup.TargetGroupArn,
			Targets:        instancesToBeRemoved,
		})
	}
	return nil
}
func (thisSwitcher *InstancesSwitcher) cleanSourceTargetGroup() {
	elbService := elbv2.New(thisSwitcher.awsSession)
	elbService.DeregisterTargets(&elbv2.DeregisterTargetsInput{
		TargetGroupArn: thisSwitcher.SelectedSourceTargetGroups[0].TargetGroupArn,
		Targets:        thisSwitcher.sourceInstancesDescriptions,
	})
}
func mapInstancesByID(instances []*elbv2.TargetDescription) map[string]int {
	result := make(map[string]int, 0)
	for _, instanceData := range instances {
		result[*instanceData.Id]++
	}
	return result
}

func (thisSwitcher *InstancesSwitcher) getInstancesInGroupsIfNeeded() error {
	if len(thisSwitcher.SelectedSourceTargetGroups) == 0 && len(thisSwitcher.SelectedTargetTargetGroups) == 0 {
		return thisSwitcher.getInstancesInGroups()
	}
	return nil
}

func (thisSwitcher *InstancesSwitcher) getInstancesInGroups() error {
	thisSwitcher.sourceInstancesDescriptions = make([]*elbv2.TargetDescription, 0)
	if err := getInstancesDescriptionForGroups(thisSwitcher.SelectedSourceTargetGroups, &thisSwitcher.sourceInstancesDescriptions, thisSwitcher.awsSession); err != nil {
		return err
	}
	var targetGroupDescription []*elbv2.TargetDescription
	for _, targetGroup := range thisSwitcher.SelectedTargetTargetGroups {
		if err := getInstancesDescriptionForGroups([]*elbv2.TargetGroup{targetGroup},
			&targetGroupDescription,
			thisSwitcher.awsSession); err != nil {
			return err
		}
		thisSwitcher.targetInstancesMapDescriptions[*targetGroup.TargetGroupName] = targetGroupDescription
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
