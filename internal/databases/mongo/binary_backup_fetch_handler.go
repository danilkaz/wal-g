package mongo

import (
	"context"
	"fmt"

	"github.com/wal-g/wal-g/internal"
	"github.com/wal-g/wal-g/internal/databases/mongo/binary"
	"github.com/wal-g/wal-g/utility"
)

func HandleBinaryFetchPush(
	ctx context.Context,
	mongodConfigPath, minimalConfigPath, backupName, restoreMongodVersion, rsName string,
	rsMembers []string,
	rsMemberIDs []int,
	shardName, mongoCfgConnectionString string,
	shardConnectionStrings []string,
	skipBackupDownload, skipReconfig, skipChecks bool,
) error {
	config, err := binary.CreateMongodConfig(mongodConfigPath)
	if err != nil {
		return err
	}

	localStorage := binary.CreateLocalStorage(config.GetDBPath())

	uploader, err := internal.ConfigureUploader()
	if err != nil {
		return err
	}
	uploader.ChangeDirectory(utility.BaseBackupPath + "/")

	if minimalConfigPath == "" {
		minimalConfigPath, err = config.SaveConfigToTempFile("storage", "systemLog")
		if err != nil {
			return err
		}
	}

	restoreService, err := binary.CreateRestoreService(ctx, localStorage, uploader, minimalConfigPath)
	if err != nil {
		return err
	}

	rsConfig := binary.NewRsConfig(rsName, rsMembers, rsMemberIDs)
	if err = rsConfig.Validate(); err != nil {
		return err
	}
	shConfig := binary.NewShConfig(shardName, mongoCfgConnectionString)
	if err = shConfig.Validate(); err != nil {
		return fmt.Errorf("ShConfig validation failed: %v", err)
	}
	mongocfgConfig, err := binary.NewMongoCfgConfig(shardConnectionStrings)
	if err != nil {
		return err
	}
	// check backup existence and resolve flag LATEST
	backup, err := internal.GetBackupByName(backupName, "", uploader.Folder())
	if err != nil {
		return err
	}

	return restoreService.DoRestore(binary.RestoreArgs{
		RsConfig:       rsConfig,
		ShConfig:       shConfig,
		MongoCfgConfig: mongocfgConfig,
		BackupName:     backup.Name,
		RestoreVersion: restoreMongodVersion,

		SkipChecks:         skipChecks,
		SkipBackupDownload: skipBackupDownload,
		SkipMongoReconfig:  skipReconfig,
	})
}
