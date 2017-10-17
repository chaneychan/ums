/*
 Navicat Premium Data Transfer

 Source Server         : ums
 Source Server Type    : MySQL
 Source Server Version : 50718
 Source Host           : localhost
 Source Database       : ums

 Target Server Type    : MySQL
 Target Server Version : 50718
 File Encoding         : utf-8

 Date: 08/24/2017 15:38:18 PM
*/

SET NAMES utf8;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
--  Table structure for `userinfo`
-- ----------------------------
DROP TABLE IF EXISTS `userinfo`;
CREATE TABLE `userinfo` (
  `id` int(10) NOT NULL AUTO_INCREMENT,
  `name` varchar(64) DEFAULT NULL,
  `password` varchar(64) DEFAULT NULL,
  `profile` varchar(64) DEFAULT NULL,
  `nickname` varchar(64) DEFAULT NULL,
  `createtime` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8;

SET FOREIGN_KEY_CHECKS = 1;
