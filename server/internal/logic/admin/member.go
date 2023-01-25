// Package admin
// @Link  https://github.com/bufanyun/hotgo
// @Copyright  Copyright (c) 2022 HotGo CLI
// @Author  Ms <133814250@qq.com>
// @License  https://github.com/bufanyun/hotgo/blob/master/LICENSE
//
package admin

import (
	"context"
	"github.com/gogf/gf/v2/crypto/gmd5"
	"github.com/gogf/gf/v2/encoding/gbase64"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/gogf/gf/v2/util/grand"
	"hotgo/api/backend/member"
	"hotgo/internal/consts"
	"hotgo/internal/dao"
	"hotgo/internal/library/contexts"
	"hotgo/internal/library/jwt"
	"hotgo/internal/library/location"
	"hotgo/internal/model"
	"hotgo/internal/model/do"
	"hotgo/internal/model/entity"
	"hotgo/internal/model/input/adminin"
	"hotgo/internal/service"
	"hotgo/utility/encrypt"
	"hotgo/utility/tree"
	"hotgo/utility/validate"
)

type sAdminMember struct{}

func NewAdminMember() *sAdminMember {
	return &sAdminMember{}
}

func init() {
	service.RegisterAdminMember(NewAdminMember())
}

// UpdateProfile 更新会员资料
func (s *sAdminMember) UpdateProfile(ctx context.Context, in adminin.MemberUpdateProfileInp) (err error) {
	memberId := contexts.Get(ctx).User.Id
	if memberId <= 0 {
		err = gerror.New("获取用户信息失败！")
		return err
	}

	var memberInfo entity.AdminMember
	if err = dao.AdminMember.Ctx(ctx).Where("id", memberId).Scan(&memberInfo); err != nil {
		err = gerror.Wrap(err, consts.ErrorORM)
		return err
	}

	_, err = dao.AdminMember.Ctx(ctx).
		Where("id", memberId).
		Data(g.Map{
			"mobile":    in.Mobile,
			"email":     in.Email,
			"real_name": in.Realname,
		}).
		Update()

	if err != nil {
		err = gerror.Wrap(err, consts.ErrorORM)
		return err
	}

	return
}

// UpdatePwd 修改登录密码
func (s *sAdminMember) UpdatePwd(ctx context.Context, in adminin.MemberUpdatePwdInp) (err error) {
	var memberInfo entity.AdminMember
	if err = dao.AdminMember.Ctx(ctx).Where("id", in.Id).Scan(&memberInfo); err != nil {
		err = gerror.Wrap(err, consts.ErrorORM)
		return err
	}

	if gmd5.MustEncryptString(in.OldPassword+memberInfo.Salt) != memberInfo.PasswordHash {
		err = gerror.New("原密码不正确")
		return err
	}

	_, err = dao.AdminMember.Ctx(ctx).
		Where("id", in.Id).
		Data(g.Map{
			"password_hash": gmd5.MustEncryptString(in.NewPassword + memberInfo.Salt),
			"updated_at":    gtime.Now(),
		}).
		Update()
	if err != nil {
		err = gerror.Wrap(err, consts.ErrorORM)
		return err
	}

	return
}

// ResetPwd 重置密码
func (s *sAdminMember) ResetPwd(ctx context.Context, in adminin.MemberResetPwdInp) (err error) {
	var (
		memberInfo entity.AdminMember
		memberId   = contexts.GetUserId(ctx)
	)
	if err = dao.AdminMember.Ctx(ctx).Where("id", in.Id).Scan(&memberInfo); err != nil {
		err = gerror.Wrap(err, consts.ErrorORM)
		return err
	}
	if memberInfo.Pid != memberId && !s.VerifySuperId(ctx, memberId) {
		err = gerror.New("操作非法")
		return err
	}

	_, err = dao.AdminMember.Ctx(ctx).
		Where("id", in.Id).
		Data(g.Map{
			"password_hash": gmd5.MustEncryptString(in.Password + memberInfo.Salt),
			"updated_at":    gtime.Now(),
		}).
		Update()
	if err != nil {
		err = gerror.Wrap(err, consts.ErrorORM)
		return err
	}

	return
}

// EmailUnique 菜单名称是否唯一
func (s *sAdminMember) EmailUnique(ctx context.Context, in adminin.MemberEmailUniqueInp) (*adminin.MemberEmailUniqueModel, error) {
	var res adminin.MemberEmailUniqueModel
	isUnique, err := dao.AdminMember.IsUniqueEmail(ctx, in.Id, in.Email)
	if err != nil {
		err = gerror.Wrap(err, consts.ErrorORM)
		return nil, err
	}

	res.IsUnique = isUnique
	return &res, nil
}

// MobileUnique 手机号是否唯一
func (s *sAdminMember) MobileUnique(ctx context.Context, in adminin.MemberMobileUniqueInp) (*adminin.MemberMobileUniqueModel, error) {
	var res adminin.MemberMobileUniqueModel
	isUnique, err := dao.AdminMember.IsUniqueMobile(ctx, in.Id, in.Mobile)
	if err != nil {
		err = gerror.Wrap(err, consts.ErrorORM)
		return nil, err
	}

	res.IsUnique = isUnique
	return &res, nil
}

// NameUnique 菜单名称是否唯一
func (s *sAdminMember) NameUnique(ctx context.Context, in adminin.MemberNameUniqueInp) (*adminin.MemberNameUniqueModel, error) {
	var res adminin.MemberNameUniqueModel
	isUnique, err := dao.AdminMember.IsUniqueName(ctx, in.Id, in.Username)
	if err != nil {
		err = gerror.Wrap(err, consts.ErrorORM)
		return nil, err
	}

	res.IsUnique = isUnique
	return &res, nil
}

// VerifySuperId 验证是否为超管
func (s *sAdminMember) VerifySuperId(ctx context.Context, verifyId int64) bool {
	superIds := g.Cfg().MustGet(ctx, "hotgo.admin.superIds")
	for _, id := range superIds.Int64s() {
		if id == verifyId {
			return true
		}
	}
	return false
}

// Delete 删除
func (s *sAdminMember) Delete(ctx context.Context, in adminin.MemberDeleteInp) error {
	if s.VerifySuperId(ctx, gconv.Int64(in.Id)) {
		return gerror.New("超管账号禁止删除！")
	}

	memberId := contexts.GetUserId(ctx)
	if memberId <= 0 {
		return gerror.New("获取用户信息失败！")
	}

	_, err := dao.AdminMember.Ctx(ctx).Where("id", in.Id).Where("pid", memberId).Delete()
	if err != nil {
		err = gerror.Wrap(err, consts.ErrorORM)
		return err
	}

	return nil
}

// Edit 修改/新增
func (s *sAdminMember) Edit(ctx context.Context, in adminin.MemberEditInp) (err error) {
	opMemberId := contexts.GetUserId(ctx)
	if opMemberId <= 0 {
		return gerror.New("获取用户信息失败！")
	}
	if in.Username == "" {
		return gerror.New("帐号不能为空")
	}

	uniqueName, err := dao.AdminMember.IsUniqueName(ctx, in.Id, in.Username)
	if err != nil {
		return gerror.Wrap(err, consts.ErrorORM)
	}
	if !uniqueName {
		return gerror.New("用户名已存在")
	}

	if in.Mobile != "" {
		uniqueMobile, err := dao.AdminMember.IsUniqueMobile(ctx, in.Id, in.Mobile)
		if err != nil {
			return gerror.Wrap(err, consts.ErrorORM)
		}
		if !uniqueMobile {
			return gerror.New("手机号已存在")
		}
	}

	if in.Email != "" {
		uniqueEmail, err := dao.AdminMember.IsUniqueEmail(ctx, in.Id, in.Email)
		if err != nil {
			return gerror.Wrap(err, consts.ErrorORM)
		}
		if !uniqueEmail {
			return gerror.New("邮箱已存在")
		}
	}

	// 修改
	if in.Id > 0 {
		if s.VerifySuperId(ctx, in.Id) {
			return gerror.New("超管账号禁止编辑！")
		}

		// 权限验证
		var mm = dao.AdminMember.Ctx(ctx).Where("id", in.Id)
		if !s.VerifySuperId(ctx, opMemberId) {
			mm = mm.Where("pid", opMemberId)
		}

		_, err = mm.Data(in).Update()
		if err != nil {
			return gerror.Wrap(err, consts.ErrorORM)
		}

		// 更新岗位
		if err = dao.AdminMemberPost.UpdatePostIds(ctx, in.Id, in.PostIds); err != nil {
			return err
		}
		return nil
	}

	// 新增用户时的额外属性
	var data adminin.MemberAddInp
	data.MemberEditInp = in
	data.Salt = grand.S(6)
	data.PasswordHash = gmd5.MustEncryptString(data.Password + data.Salt)

	// 关系树
	data.Pid = opMemberId
	data.Level, data.Tree, err = s.genTree(ctx, opMemberId)
	if err != nil {
		return err
	}

	id, err := dao.AdminMember.Ctx(ctx).Data(data).InsertAndGetId()
	if err != nil {
		err = gerror.Wrap(err, consts.ErrorORM)
		return err
	}

	err = dao.AdminMemberPost.UpdatePostIds(ctx, id, in.PostIds)
	if err != nil {
		return err
	}
	return nil
}

// MaxSort 最大排序
func (s *sAdminMember) MaxSort(ctx context.Context, in adminin.MemberMaxSortInp) (*adminin.MemberMaxSortModel, error) {
	var res adminin.MemberMaxSortModel

	if in.Id > 0 {
		if err := dao.AdminMember.Ctx(ctx).Where("id", in.Id).Order("sort desc").Scan(&res); err != nil {
			err = gerror.Wrap(err, consts.ErrorORM)
			return nil, err
		}
	}

	res.Sort = res.Sort + 10

	return &res, nil
}

// View 获取信息
func (s *sAdminMember) View(ctx context.Context, in adminin.MemberViewInp) (res *adminin.MemberViewModel, err error) {
	if err = dao.AdminMember.Ctx(ctx).Where("id", in.Id).Scan(&res); err != nil {
		err = gerror.Wrap(err, consts.ErrorORM)
		return nil, err
	}

	return res, nil
}

// List 获取列表
func (s *sAdminMember) List(ctx context.Context, in adminin.MemberListInp) (list []*adminin.MemberListModel, totalCount int, err error) {
	g.Log().Printf(ctx, "in:%#v", in)
	mod := dao.AdminMember.Ctx(ctx)
	if in.Realname != "" {
		mod = mod.WhereLike("real_name", "%"+in.Realname+"%")
	}
	if in.Username != "" {
		mod = mod.WhereLike("username", "%"+in.Username+"%")
	}
	if in.Mobile > 0 {
		mod = mod.Where("mobile", in.Mobile)
	}
	if in.Status > 0 {
		mod = mod.Where("status", in.Status)
	}
	if in.DeptId > 0 {
		mod = mod.Where("dept_id", in.DeptId)
	}

	if len(in.CreatedAt) == 2 {
		mod = mod.WhereBetween("created_at", gtime.New(in.CreatedAt[0]), gtime.New(in.CreatedAt[1]))
	}

	totalCount, err = mod.Count()
	if err != nil {
		err = gerror.Wrap(err, consts.ErrorORM)
		return nil, 0, err
	}

	if totalCount == 0 {
		return list, totalCount, nil
	}

	if err = mod.Page(in.Page, in.PerPage).Order("id desc").Scan(&list); err != nil {
		return nil, 0, gerror.Wrap(err, consts.ErrorORM)
	}

	// 重写树入参
	for i := 0; i < len(list); i++ {
		// 部门
		deptName, err := dao.AdminDept.Ctx(ctx).
			Fields("name").
			Where("id", list[i].DeptId).
			Value()
		if err != nil {
			return nil, 0, gerror.Wrap(err, consts.ErrorORM)
		}
		list[i].DeptName = deptName.String()

		// 角色
		roleName, err := dao.AdminRole.Ctx(ctx).
			Fields("name").
			Where("id", list[i].RoleId).
			Value()
		if err != nil {
			return nil, 0, gerror.Wrap(err, consts.ErrorORM)
		}
		list[i].RoleName = roleName.String()

		// 岗位
		posts, err := dao.AdminMemberPost.Ctx(ctx).
			Fields("post_id").
			Where("member_id", list[i].Id).
			Array()
		if err != nil {
			return nil, 0, gerror.Wrap(err, consts.ErrorORM)
		}

		for _, v := range posts {
			list[i].PostIds = append(list[i].PostIds, v.Int64())
		}
	}

	return list, totalCount, nil
}

// genTree 生成关系树
func (s *sAdminMember) genTree(ctx context.Context, pid int64) (level int, newTree string, err error) {
	var (
		pInfo *entity.AdminMember
	)
	err = dao.AdminMember.Ctx(ctx).Where("id", pid).Scan(&pInfo)
	if err != nil {
		return
	}

	if pInfo == nil {
		err = gerror.New("上级信息不存在")
		return
	}

	level = pInfo.Level + 1
	newTree = tree.GenLabel(pInfo.Tree, pInfo.Id)

	return
}

// LoginMemberInfo 获取登录用户信息
func (s *sAdminMember) LoginMemberInfo(ctx context.Context, req *member.InfoReq) (res *adminin.MemberLoginModel, err error) {
	identity := contexts.Get(ctx).User
	if identity == nil {
		err = gerror.New("用户身份异常，请重新登录！")
		return
	}

	permissions, err := service.AdminMenu().LoginPermissions(ctx, identity.Id)
	if err != nil {
		return
	}

	res = &adminin.MemberLoginModel{
		UserId:      identity.Id,
		Username:    identity.Username,
		RealName:    identity.RealName,
		Avatar:      identity.Avatar,
		Permissions: permissions,
		Token:       jwt.GetAuthorization(ghttp.RequestFromCtx(ctx)),
	}

	return
}

// Login 提交登录
func (s *sAdminMember) Login(ctx context.Context, in adminin.MemberLoginInp) (res *adminin.MemberLoginModel, err error) {
	var (
		roleInfo   *entity.AdminRole
		memberInfo *entity.AdminMember
		identity   *model.Identity
		timestamp  = gtime.Timestamp()
		expires    = g.Cfg().MustGet(ctx, "jwt.expires", 1).Int64()
		exp        = gconv.Int64(timestamp) + expires
		lastIp     = location.GetClientIp(ghttp.RequestFromCtx(ctx))
	)

	err = dao.AdminMember.Ctx(ctx).Where("username", in.Username).Scan(&memberInfo)
	if err != nil {
		err = gerror.Wrap(err, consts.ErrorORM)
		return
	}
	if memberInfo == nil {
		err = gerror.New("账号不存在")
		return
	}

	if memberInfo.Salt == "" {
		err = gerror.New("用户信息错误")
		return
	}

	// 解密密码
	password, err := gbase64.Decode([]byte(in.Password))
	if err != nil {
		return nil, err
	}
	password, err = encrypt.AesECBDecrypt(password, consts.RequestEncryptKey)
	if err != nil {
		return nil, err
	}

	if memberInfo.PasswordHash != gmd5.MustEncryptString(string(password)+memberInfo.Salt) {
		err = gerror.New("用户密码不正确")
		return
	}

	err = dao.AdminRole.Ctx(ctx).
		Fields("id,key,status").
		Where("id", memberInfo.RoleId).
		Scan(&roleInfo)
	if err != nil {
		err = gerror.Wrap(err, consts.ErrorORM)
		return
	}
	if roleInfo == nil {
		err = gerror.New("角色不存在")
		return
	}

	if roleInfo.Status != consts.StatusEnabled {
		err = gerror.New("角色权限已被禁用")
		return
	}

	identity = &model.Identity{
		Id:         memberInfo.Id,
		Pid:        memberInfo.Pid,
		DeptId:     memberInfo.DeptId,
		RoleId:     roleInfo.Id,
		RoleKey:    roleInfo.Key,
		Username:   memberInfo.Username,
		RealName:   memberInfo.RealName,
		Avatar:     memberInfo.Avatar,
		Email:      memberInfo.Email,
		Mobile:     memberInfo.Mobile,
		VisitCount: memberInfo.VisitCount,
		LastTime:   memberInfo.LastTime,
		LastIp:     lastIp,
		Exp:        exp,
		Expires:    expires,
		App:        consts.AppAdmin,
	}

	token, err := jwt.GenerateLoginToken(ctx, identity, false)
	if err != nil {
		err = gerror.New(err.Error())
		return
	}

	// 更新登录信息
	_, err = dao.AdminMember.Ctx(ctx).Data(do.AdminMember{
		AuthKey:    gmd5.MustEncryptString(token),
		VisitCount: memberInfo.VisitCount + 1,
		LastTime:   timestamp,
		LastIp:     lastIp,
	}).Where(do.AdminMember{
		Id: memberInfo.Id,
	}).Update()

	if err != nil {
		err = gerror.New(err.Error())
		return
	}

	res = &adminin.MemberLoginModel{
		UserId:   identity.Id,
		Username: identity.Username,
		RealName: identity.RealName,
		Avatar:   identity.Avatar,
		Token:    token,
	}

	return res, nil
}

// RoleMemberList 获取角色下的会员列表
func (s *sAdminMember) RoleMemberList(ctx context.Context, in adminin.RoleMemberListInp) (list []*adminin.MemberListModel, totalCount int, err error) {
	mod := dao.AdminMember.Ctx(ctx)
	if in.Role > 0 {
		mod = mod.Where("role", in.Role)
	}

	totalCount, err = mod.Count()
	if err != nil {
		err = gerror.Wrap(err, consts.ErrorORM)
		return list, totalCount, err
	}

	err = mod.Page(in.Page, in.PerPage).Order("id desc").Scan(&list)
	if err != nil {
		err = gerror.Wrap(err, consts.ErrorORM)
		return list, totalCount, err
	}

	return list, totalCount, err
}

// Status 更新状态
func (s *sAdminMember) Status(ctx context.Context, in adminin.MemberStatusInp) (err error) {
	if in.Id <= 0 {
		err = gerror.New("ID不能为空")
		return err
	}

	if s.VerifySuperId(ctx, in.Id) {
		return gerror.New("超管账号不能更改状态")
	}

	if in.Status <= 0 {
		err = gerror.New("状态不能为空")
		return err
	}

	if !validate.InSliceInt(consts.StatusMap, in.Status) {
		err = gerror.New("状态不正确")
		return err
	}

	// 修改
	in.UpdatedAt = gtime.Now()
	_, err = dao.AdminMember.Ctx(ctx).Where("id", in.Id).Data("status", in.Status).Update()
	if err != nil {
		err = gerror.Wrap(err, consts.ErrorORM)
		return err
	}

	return nil
}

// GetIdByCode 通过邀请码获取会员ID
func (s *sAdminMember) GetIdByCode(ctx context.Context, in adminin.GetIdByCodeInp) (res *adminin.GetIdByCodeModel, err error) {
	if err = dao.AdminMember.Ctx(ctx).
		Fields("invite_code").
		Where("invite_code", in.Code).
		Scan(&res); err != nil {
		err = gerror.Wrap(err, consts.ErrorORM)
		return nil, err
	}

	return res, nil
}
