/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validation

import (
	corev1 "k8s.io/api/core/v1"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	coapi "github.com/openshift/cluster-operator/pkg/apis/clusteroperator"
)

// ValidateClusterDeploymentName validates the name of a cluster.
var ValidateClusterDeploymentName = apivalidation.ValidateClusterName

// ValidateClusterDeployment validates a cluster being created.
func ValidateClusterDeployment(cluster *coapi.ClusterDeployment) field.ErrorList {
	allErrs := field.ErrorList{}

	for _, msg := range ValidateClusterDeploymentName(cluster.Name, false) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("name"), cluster.Name, msg))
	}
	allErrs = append(allErrs, validateClusterDeploymentSpec(&cluster.Spec, field.NewPath("spec"))...)
	allErrs = append(allErrs, validateClusterDeploymentStatus(&cluster.Status, field.NewPath("status"))...)

	return allErrs
}

// validateClusterDeploymentSpec validates the spec of a cluster.
func validateClusterDeploymentSpec(spec *coapi.ClusterDeploymentSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if spec.ClusterName == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("clusterName"), "ClusterName must be set"))
	}
	machineSetsPath := fldPath.Child("machineSets")
	versionPath := fldPath.Child("clusterVersionRef")
	masterCount := 0
	infraCount := 0
	machineSetShortNames := map[string]bool{}
	for i := range spec.MachineSets {
		machineSet := spec.MachineSets[i]
		allErrs = append(allErrs, validateClusterMachineSet(&machineSet, machineSetsPath.Index(i))...)
		if machineSet.NodeType == coapi.NodeTypeMaster {
			masterCount++
			if masterCount > 1 {
				allErrs = append(allErrs, field.Invalid(machineSetsPath.Index(i).Child("type"), machineSet.NodeType, "can only have one master machineset"))
			}
		} else {
			if machineSetShortNames[machineSet.ShortName] {
				allErrs = append(allErrs, field.Duplicate(machineSetsPath.Index(i).Child("shortName"), machineSet.ShortName))
			}
			machineSetShortNames[machineSet.ShortName] = true
		}

		if machineSet.Infra {
			infraCount++
			if infraCount > 1 {
				allErrs = append(allErrs, field.Invalid(machineSetsPath.Index(i).Child("infra"), machineSet.Infra, "can only have one infra machineset"))
			}
		}
	}

	if masterCount == 0 {
		allErrs = append(allErrs, field.Invalid(machineSetsPath, &spec.MachineSets, "must have one machineset with a master node type"))
	}

	if infraCount == 0 {
		allErrs = append(allErrs, field.Invalid(machineSetsPath, &spec.MachineSets, "must have one machineset that hosts infra pods"))
	}

	if len(spec.Config.SDNPluginName) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("config").Child("sdnPluginName"), "must specify an SDN plugin name to install"))
	}

	if len(spec.ClusterVersionRef.Name) == 0 {
		allErrs = append(allErrs, field.Required(versionPath.Child("name"), "must specify a cluster version to install"))
	}

	if spec.NetworkConfig.ServiceDomain == "" {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec", "networkConfig", "serviceDomain"),
			spec.NetworkConfig.ServiceDomain,
			"invalid cluster configuration: missing service domain"))
	}

	if len(spec.NetworkConfig.Pods.CIDRBlocks) == 0 {
		allErrs = append(allErrs, field.Invalid(
			fldPath.Child("networkConfig", "pods"),
			spec.NetworkConfig.Pods,
			"missing cluster network POD CIDR"))
	}
	if len(spec.NetworkConfig.Services.CIDRBlocks) == 0 {
		allErrs = append(allErrs, field.Invalid(
			fldPath.Child("networkConfig", "services"),
			spec.NetworkConfig.Services,
			"missing cluster network services CIDR"))
	}

	if len(spec.NetworkConfig.Services.CIDRBlocks) > 1 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("networkConfig").Child("services"), &spec.NetworkConfig.Services, "only one service network subnet is currently supported"))
	}
	if len(spec.NetworkConfig.Pods.CIDRBlocks) > 1 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("networkConfig").Child("pods"), &spec.NetworkConfig.Pods, "only one pod network subnet is currently supported"))
	}

	return allErrs
}

func validateClusterMachineSet(machineSet *coapi.ClusterMachineSet, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if machineSet.NodeType == coapi.NodeTypeMaster {
		if len(machineSet.ShortName) > 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("shortName"), machineSet.ShortName, "short name must not be specified for master machineset"))
		}
	} else {
		if machineSet.ShortName == coapi.MasterMachineSetName {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("shortName"), machineSet.ShortName, "short name cannot be reserved name"))
		}
		for _, msg := range apivalidation.NameIsDNSLabel(machineSet.ShortName, false) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("shortName"), machineSet.ShortName, msg))
		}
	}
	allErrs = append(allErrs, validateMachineSetConfig(&machineSet.MachineSetConfig, fldPath)...)
	return allErrs
}

// validateClusterDeploymentStatus validates the status of a cluster.
func validateClusterDeploymentStatus(status *coapi.ClusterDeploymentStatus, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	return allErrs
}

func validateSecretRef(ref *corev1.LocalObjectReference, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if ref == nil {
		return allErrs
	}
	if len(ref.Name) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "must specify a secret name"))
	}
	return allErrs
}

// ValidateClusterDeploymentUpdate validates an update to the spec of a cluster.
func ValidateClusterDeploymentUpdate(new *coapi.ClusterDeployment, old *coapi.ClusterDeployment) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateClusterDeploymentSpec(&new.Spec, field.NewPath("spec"))...)

	allErrs = append(allErrs, apivalidation.ValidateImmutableField(new.Spec.ClusterName, old.Spec.ClusterName, field.NewPath("spec", "clusterName"))...)
	allErrs = append(allErrs, apivalidation.ValidateImmutableField(new.Spec.NetworkConfig.Services, old.Spec.NetworkConfig.Services, field.NewPath("spec", "networkConfig", "services"))...)
	allErrs = append(allErrs, apivalidation.ValidateImmutableField(new.Spec.NetworkConfig.Pods, old.Spec.NetworkConfig.Pods, field.NewPath("spec", "networkConfig", "pods"))...)
	allErrs = append(allErrs, apivalidation.ValidateImmutableField(new.Spec.Config.SDNPluginName, old.Spec.Config.SDNPluginName, field.NewPath("spec", "config", "sdnPluginName"))...)

	return allErrs
}

// ValidateClusterDeploymentStatusUpdate validates an update to the status of a cluster.
func ValidateClusterDeploymentStatusUpdate(new *coapi.ClusterDeployment, old *coapi.ClusterDeployment) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateClusterDeploymentStatus(&new.Status, field.NewPath("status"))...)

	return allErrs
}
