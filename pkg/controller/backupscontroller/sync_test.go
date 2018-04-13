/*
Copyright 2018 Pressinfra SRL

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

package backupscontroller

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/record"

	fakeMyClient "github.com/presslabs/mysql-operator/pkg/generated/clientset/versioned/fake"
	informers "github.com/presslabs/mysql-operator/pkg/generated/informers/externalversions"
	tutil "github.com/presslabs/mysql-operator/pkg/util/test"
)

func newController(stop chan struct{}, client *fake.Clientset,
	myClient *fakeMyClient.Clientset,
	rec *record.FakeRecorder,
) *Controller {

	sharedInformerFactory := informers.NewSharedInformerFactory(
		myClient, time.Second)
	kubeSharedInformerFactory := kubeinformers.NewSharedInformerFactory(
		client, time.Second)

	sharedInformerFactory.Start(stop)
	kubeSharedInformerFactory.Start(stop)

	return New(
		client,
		myClient,
		sharedInformerFactory.Mysql().V1alpha1().MysqlBackups(),
		sharedInformerFactory.Mysql().V1alpha1().MysqlClusters(),
		rec,
		tutil.Namespace,
		kubeSharedInformerFactory.Batch().V1().Jobs(),
	)
}

// TestBackupCompleteSync
// Test: a backup already  completed
// Expect: skip sync-ing
func TestBackupCompleteSync(t *testing.T) {
	client := fake.NewSimpleClientset()
	myClient := fakeMyClient.NewSimpleClientset()
	rec := record.NewFakeRecorder(100)

	stop := make(chan struct{})
	defer close(stop)
	controller := newController(stop, client, myClient, rec)

	cluster := tutil.NewFakeCluster("asd")
	_, err := myClient.MysqlV1alpha1().MysqlClusters(tutil.Namespace).Create(cluster)
	if err != nil {
		fmt.Println("Failed to create cluster:", err)
	}
	backup := tutil.NewFakeBackup("asd-backup", cluster.Name)
	backup.Status.Completed = true

	ctx := context.TODO()
	err = controller.Sync(ctx, backup, tutil.Namespace)
	if err != nil {
		fmt.Println("Sync err: ", err)
		t.Fail()
	}
}

// TestBackupSyncNoClusterName
// Test: backup without cluster name
// Expect: sync to fail
func TestBackupSyncNoClusterName(t *testing.T) {
	client := fake.NewSimpleClientset()
	myClient := fakeMyClient.NewSimpleClientset()
	rec := record.NewFakeRecorder(100)

	stop := make(chan struct{})
	defer close(stop)
	controller := newController(stop, client, myClient, rec)

	backup := tutil.NewFakeBackup("asd-backup", "")
	backup.Status.Completed = true

	ctx := context.TODO()
	err := controller.Sync(ctx, backup, tutil.Namespace)
	if !strings.Contains(err.Error(), "cluster name is not specified") {
		t.Fail()
	}
}
