#!/bin/bash

git pull
go install -v
systemctl restart reminderbot.service
journalctl -fu reminderbot.service