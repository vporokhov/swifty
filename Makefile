.PHONY: all .FORCE
.DEFAULT_GOAL := all

include Makefile.inc
include Makefile.versions

GITID_FILE	:= .gitid
GITID		:= $(shell if [ -d ".git" ]; then git describe --always; fi)
ifeq ($(GITID),)
        GITID := 0
else
        GITID_FILE_VALUE := $(shell if [ -f '$(GITID_FILE)' ]; then if [ `cat '$(GITID_FILE)'` = $(GITID) ]; then echo y; fi; fi)
        ifneq ($(GITID_FILE_VALUE),y)
                .PHONY: $(GITID_FILE)
        endif
endif

export GITID

$(GITID_FILE):
	$(call msg-gen, $@)
	$(Q) echo "$(GITID)" > $(GITID_FILE)

# Build daemon
define gen-gobuild
swy-$(1): $$(go-$(1)-y) .FORCE
	$$(call msg-gen,$$@)
	$$(Q) $$(GO) $$(GO-BUILD-OPTS) -o $$@ $$(go-$(1)-y)
all-y += swy-$(1)
endef

# Build native
define gen-gobuild-n
src/$(1)/version.go: .FORCE
	$$(call msg-gen, $$@)
	$$(Q) echo '// Autogenerated'					>  $$@
	$$(Q) echo 'package main' 					>> $$@
	$$(Q) echo 'var Version string = "$(GATE_VERSION)-$(GITID)"'	>> $$@
	$$(Q) echo 'var Flavor string = "$(FLAVOR)"'			>> $$@

swy-$(1): .FORCE src/$(1)/version.go
	$$(call msg-gen,$$@)
	$$(Q) cd src/$(1)/ && go build
	$$(Q) $$(MV) src/$(1)/$(1) $$@
all-y += swy-$(1)
endef

# Build tool
define gen-gobuild-t
swy$(1): $$(go-$(1)-y) .FORCE
	$$(call msg-gen,$$@)
	$$(Q) $$(GO) $$(GO-BUILD-OPTS) -o $$@ $$(go-$(1)-y)
all-y += swy$(1)
endef

go-pgrest-y	+= src/pgrest/main.go
go-mquotad-y	+= src/mquotad/main.go
go-ctl-y	+= src/tools/ctl.go
go-trace-y	+= src/tools/tracer.go
go-s3fsck-y	+= src/tools/s3-fsck.go
go-sg-y		+= src/tools/sg.go
go-dbscr-y	+= src/tools/scraper.go

$(eval $(call gen-gobuild-n,gate))
$(eval $(call gen-gobuild-n,admd))
$(eval $(call gen-gobuild-n,s3))
$(eval $(call gen-gobuild-n,wdog))
$(eval $(call gen-gobuild-n,mongoproxy))
#$(eval $(call gen-gobuild,pgrest))
#$(eval $(call gen-gobuild,mquotad))
$(eval $(call gen-gobuild-t,ctl))
$(eval $(call gen-gobuild-t,trace))
$(eval $(call gen-gobuild-t,s3fsck))
$(eval $(call gen-gobuild-t,sg))
$(eval $(call gen-gobuild-t,dbscr))

# Default target
all: $(all-y)

swy-runner: src/wdog/runner/runner.c
	$(call msg-gen,$@)
	$(Q) $(CC) -Wall -Werror -O2 -static -o $@ $<

LANGS = python golang swift ruby nodejs csharp
IMAGES =

define gen-lang
IMAGES += swifty/$(1)
swifty/$(1): swy-wdog swy-runner kubectl/docker/wdog/$(1)/Dockerfile
endef

$(foreach l,$(LANGS),$(eval $(call gen-lang,$l)))

#
# Docker images
swifty/python: src/wdog/runner/runner.py
	$(call msg-gen,$@)
	$(Q) $(CP) swy-wdog  kubectl/docker/wdog/python/swy-wdog
	$(Q) $(CP) swy-runner  kubectl/docker/wdog/python/
	$(Q) $(CP) src/wdog/runner/runner.py  kubectl/docker/wdog/python/swy-runner.py
	$(Q) $(CP) src/wdog/lib/lib.py kubectl/docker/wdog/python/swifty.py
	$(Q) $(MAKE) -C kubectl/docker/wdog/python all
.PHONY: swifty/python

swifty/golang: src/wdog/runner/runner.go
	$(call msg-gen,$@)
	$(Q) $(CP) swy-wdog  kubectl/docker/wdog/golang/swy-wdog
	$(Q) $(CP) swy-runner  kubectl/docker/wdog/golang/
	$(Q) $(CP) src/wdog/runner/runner.go kubectl/docker/wdog/golang/
	$(Q) $(CP) src/wdog/runner/body.go kubectl/docker/wdog/golang/
	$(Q) $(CP) src/wdog/lib/lib.go kubectl/docker/wdog/golang/
	$(Q) $(CP) src/common/xqueue/queue.go kubectl/docker/wdog/golang/
	$(Q) $(MAKE) -C kubectl/docker/wdog/golang all
.PHONY: swifty/golang

swifty/swift: src/wdog/runner/runner.swift
	$(call msg-gen,$@)
	$(Q) $(CP) swy-wdog  kubectl/docker/wdog/swift/swy-wdog
	$(Q) $(CP) swy-runner  kubectl/docker/wdog/swift/
	$(Q) $(CP) src/wdog/runner/runner.swift kubectl/docker/wdog/swift/
	$(Q) $(MAKE) -C kubectl/docker/wdog/swift all
.PHONY: swifty/swift

src/wdog/lib/XStream.dll: src/wdog/lib/XStream.cs
	docker run --rm -v $(CURDIR)/src/wdog/lib/:/mono mono csc /mono/XStream.cs -out:/mono/XStream.dll -target:library -r:Mono.Posix.dll -unsafe

swifty/csharp: src/wdog/runner/runner.cs src/wdog/lib/XStream.dll
	$(call msg-gen,$@)
	$(Q) $(CP) swy-wdog  kubectl/docker/wdog/csharp/swy-wdog
	$(Q) $(CP) swy-runner  kubectl/docker/wdog/csharp/
	$(Q) $(CP) src/wdog/runner/runner.cs kubectl/docker/wdog/csharp/
	$(Q) $(CP) src/wdog/lib/XStream.dll kubectl/docker/wdog/csharp/
	$(Q) $(MAKE) -C kubectl/docker/wdog/csharp all
.PHONY: swifty/csharp

swifty/nodejs: src/wdog/runner/runner.js
	$(call msg-gen,$@)
	$(Q) $(CP) swy-wdog  kubectl/docker/wdog/nodejs/swy-wdog
	$(Q) $(CP) swy-runner  kubectl/docker/wdog/nodejs/
	$(Q) $(CP) src/wdog/runner/runner.js kubectl/docker/wdog/nodejs/
	$(Q) $(MAKE) -C kubectl/docker/wdog/nodejs all
.PHONY: swifty/nodejs

swifty/ruby: src/wdog/runner/runner.js
	$(call msg-gen,$@)
	$(Q) $(CP) swy-wdog kubectl/docker/wdog/ruby/swy-wdog
	$(Q) $(CP) swy-runner  kubectl/docker/wdog/ruby/
	$(Q) $(CP) src/wdog/runner/runner.rb kubectl/docker/wdog/ruby/
	$(Q) $(MAKE) -C kubectl/docker/wdog/ruby all
.PHONY: swifty/ruby

swifty/gate: swy-gate kubectl/docker/gate/Dockerfile test/functions/golang/simple-user-mgmt.go
	$(call msg-gen,$@)
	$(Q) $(CP) swy-gate kubectl/docker/gate/swy-gate
	$(Q) $(MAKE) -C kubectl/docker/gate all
.PHONY: swifty/gate

swifty/admd: swy-admd kubectl/docker/admd/Dockerfile
	$(call msg-gen,$@)
	$(Q) $(CP) swy-admd kubectl/docker/admd/swy-admd
	$(Q) $(MAKE) -C kubectl/docker/admd all
.PHONY: swifty/admd

swifty/swydbscr: swydbscr kubectl/docker/dbscr/Dockerfile
	$(call msg-gen,$@)
	$(Q) $(CP) swydbscr kubectl/docker/dbscr/swydbscr
	$(Q) $(MAKE) -C kubectl/docker/dbscr all
.PHONY: swifty/swydbscr

swifty/proxy: swy-wdog kubectl/docker/proxy/Dockerfile
	$(call msg-gen,$@)
	$(Q) $(CP) swy-wdog kubectl/docker/proxy/swy-wdog
	$(Q) $(MAKE) -C kubectl/docker/proxy all
.PHONY: swifty/proxy

swifty/s3: swy-s3 kubectl/docker/s3/Dockerfile
	$(call msg-gen,$@)
	$(Q) $(CP) swy-s3 kubectl/docker/s3/swy-s3
	$(Q) $(MAKE) -C kubectl/docker/s3 all

images: $(IMAGES)
	@true

.PHONY: images

help:
	@echo '    Targets:'
	@echo '      all             - Build all [*] targets'
	@echo '      images          - Build all docker images'
	@echo '      docs            - Build documentation'
	@echo '    * swy-gate        - Build gate'
	@echo '    * swy-wdog        - Build watchdog'
	@echo '    * swy-admd        - Build adm daemon'
#	@echo '    * swy-pgrest      - Build pgrest daemon'
#	@echo '    * swy-mquotad     - Build mquotad daemon'
	@echo '    * swy-wdog        - Build golang daemon'
	@echo '    * swy-s3          - Build s3 daemon'
	@echo '    * swyctl          - Build gate cli'
	@echo '    * swytrace        - Build gate eq tracing tool'
	@echo '    * swys3fsck       - Build s3 databaase integrity checker'
	@echo '    * swysg           - Build secrets generator cli'
	@echo '    * swydbscr        - Build DB scraper tool'
	@echo '      swifty/python   - Build swifty/python docker image'
	@echo '      swifty/golang   - Build swifty/golang docker image'
	@echo '      swifty/swift    - Build swifty/swift docker image'
	@echo '      swifty/nodejs   - Build swifty/nodejs docker image'
	@echo '      swifty/ruby     - Build swifty/ruby docker image'
	@echo '      rsclean         - Cleanup resources'
	@echo '      clean-db-swifty - Cleanup swifty mongo collections'
	@echo '      clean-db-s3     - Cleanup s3 mongo collections'
	@echo '      mqclean         - Cleanup rabbitmq'
	@echo '      sqlclean        - Cleanup mariadb'
.PHONY: help

tags:
	$(Q) $(GOTAGS) -R src/ > tags
.PHONY: tags

#docs: .FORCE
#	$(Q) $(MAKE) -C docs/ html
#.PHONY: docs

tarball:
	$(Q) $(GIT) archive --format=tar --prefix=swifty/ HEAD > swifty.tar
.PHONY: tarball

ifneq ($(filter mqclean,$(MAKECMDGOALS)),)
rabbit-users := $(filter-out guest root, $(shell rabbitmqctl list_users | tail -n +2 | cut -f 1))
rabbit-vhosts := $(filter-out /, $(shell rabbitmqctl list_vhosts | tail -n +2 | cut -f 1))
endif

mqclean: .FORCE
	$(call msg-gen,"Cleaning up MessageQ")
ifneq ($(rabbit-users),)
	$(Q) rabbitmqctl delete_user $(rabbit-users)
endif
ifneq ($(rabbit-vhosts),)
	$(Q) rabbitmqctl delete_vhost $(rabbit-vhosts)
endif
.PHONY: mqclean

ifneq ($(filter sqlclean,$(MAKECMDGOALS)),)
mysql-user ?= "root"
mysql-pass ?= "aiNe1sah9ichu1re"
sql-users := $(filter-out root, \
	$(shell mysql -u$(mysql-user) -p$(mysql-pass) -N -e'select user from mysql.user' | cut -f1))
sql-dbases := $(filter-out information_schema mysql performance_schema test, \
	$(shell mysql -u$(mysql-user) -p$(mysql-pass) -N -e'show databases' | cut -f1))
endif

sqlclean: .FORCE
	$(call msg-gen,"Cleaning up SQL")
ifneq ($(sql-users),)
	$(foreach user,$(sql-users),$(shell mysql -u$(mysql-user) -p$(mysql-pass) -e'drop user $(user)'))
endif
ifneq ($(sql-dbases),)
	$(foreach db,$(sql-dbases),$(shell mysql -u$(mysql-user) -p$(mysql-pass) -e'drop database $(db)'))
endif
.PHONY: sqlclean

DB-SWIFTY	:= swifty
DB-S3		:= swifty-s3

deps:
	sh deps.sh

mgo-swifty-creds :=
ifneq ($(mgo-swifty-user),)
ifneq ($(mgo-swifty-pass),)
	mgo-swifty-creds := -u $(mgo-swifty-user) -p $(mgo-swifty-pass)
endif
endif

mgo-s3-creds :=
ifneq ($(mgo-s3-user),)
ifneq ($(mgo-s3-pass),)
	mgo-s3-creds := -u $(mgo-s3-user) -p $(mgo-s3-pass)
endif
endif

clean-db-swifty:
	$(call msg-gen,"Cleaning up main MongoDB")
	$(Q) $(MONGO)/$(DB-SWIFTY) $(mgo-swifty-creds) --eval 'db.Function.remove({});'
	$(Q) $(MONGO)/$(DB-SWIFTY) $(mgo-swifty-creds) --eval 'db.Mware.remove({});'
	$(Q) $(MONGO)/$(DB-SWIFTY) $(mgo-swifty-creds) --eval 'db.FnStats.remove({});'
	$(Q) $(MONGO)/$(DB-SWIFTY) $(mgo-swifty-creds) --eval 'db.Pods.remove({});'
	$(Q) $(MONGO)/$(DB-SWIFTY) $(mgo-swifty-creds) --eval 'db.Balancer.remove({});'
	$(Q) $(MONGO)/$(DB-SWIFTY) $(mgo-swifty-creds) --eval 'db.BalancerRS.remove({});'
	$(Q) $(MONGO)/$(DB-SWIFTY) $(mgo-swifty-creds) --eval 'db.Logs.remove({});'
.PHONY: clean-db-swifty

clean-db-s3:
	$(call msg-gen,"Cleaning up s3 MongoDB")
	$(Q) $(MONGO)/$(DB-S3) $(mgo-s3-creds) --eval 'db.S3Iams.remove({});'
	$(Q) $(MONGO)/$(DB-S3) $(mgo-s3-creds) --eval 'db.S3Buckets.remove({});'
	$(Q) $(MONGO)/$(DB-S3) $(mgo-s3-creds) --eval 'db.S3Objects.remove({});'
	$(Q) $(MONGO)/$(DB-S3) $(mgo-s3-creds) --eval 'db.S3Uploads.remove({});'
	$(Q) $(MONGO)/$(DB-S3) $(mgo-s3-creds) --eval 'db.S3ObjectData.remove({});'
	$(Q) $(MONGO)/$(DB-S3) $(mgo-s3-creds) --eval 'db.S3AccessKeys.remove({});'
.PHONY: clean-db-s3

rsclean: clean-db-swifty clean-db-s3
	$(call msg-gen,"Cleaning up kubernetes")
	$(Q) $(KUBECTL) delete deployment --all
	$(Q) $(KUBECTL) delete secret --all
	#$(Q) $(KUBECTL) delete service --all
	$(Q) $(KUBECTL) delete pod --all
	$(call msg-gen,"Cleaning up IPVS")
	$(Q) $(IPVSADM) -C
	$(call msg-gen,"Cleaning up FS")
ifneq ($(wildcard $(LOCAL_SOURCES)/.*),)
	$(Q) $(RM) -r $(LOCAL_SOURCES)/*
endif
ifneq ($(wildcard $(VOLUME_DIR)/.*),)
	$(Q) $(RM) -r $(VOLUME_DIR)/*
endif
ifneq ($(wildcard $(TEST_REPO)/.*),)
	$(Q) $(RM) -r $(TEST_REPO)/*
endif

clean:
	$(call msg-clean,swy-gate)
	$(Q) $(RM) swy-gate
	$(call msg-clean,swy-wdog)
	$(Q) $(RM) swy-wdog
	$(call msg-clean,swy-admd)
	$(Q) $(RM) swy-admd
	$(call msg-clean,swy-pgrest)
	$(Q) $(RM) swy-pgrest
	$(call msg-clean,swy-mquotad)
	$(Q) $(RM) swy-mquotad
	$(call msg-clean,swy-wdog)
	$(Q) $(RM) swy-wdog
	$(call msg-clean,swy-s3)
	$(Q) $(RM) swy-s3
	$(call msg-clean,swyctl)
	$(Q) $(RM) swyctl
	$(call msg-clean,swytrace)
	$(Q) $(RM) swytrace
	$(call msg-clean,swydbscr)
	$(Q) $(RM) swydbscr
	$(call msg-clean,swys3fsck)
	$(Q) $(RM) swys3fsck
	$(call msg-clean,swysg)
	$(Q) $(RM) swysg
#	$(Q) $(MAKE) -C docs clean
.PHONY: clean

mrproper: clean
	$(call msg-clean,tags)
	$(Q) $(RM) tags
.PHONY: mrproper

.SUFFIXES:
