import {
    Person as PersonIcon,
    PersonAdd as PersonAddIcon,
    Settings as SettingsIcon,
    Shop as ShopIcon,
    ShoppingCart as ShoppingCartIcon
} from '@material-ui/icons';
import { ROLES } from '@local/shared';
import { LINKS } from 'utils';
import _ from 'lodash';

// Returns user actions, in a list of this format:
//  [
//      label: str,
//      value: str,
//      link: str,
//      onClick: func,
//      icon: Material-UI Icon,
//      number of notifications: int,
//  ]
export function getUserActions(session, userRoles, cart) {
    let actions = [];

    // If someone is not logged in, display sign up/log in links
    if (!_.isObject(session) || Object.entries(session).length === 0) {
        actions.push(['Log In', 'login', LINKS.LogIn, null, PersonAddIcon, 0]);
    } else {
        // If an owner admin is logged in, display owner links
        const haveArray = Array.isArray(userRoles) ? userRoles : [userRoles];
        if (userRoles && haveArray.some(r => [ROLES.Owner, ROLES.Admin].includes(r?.role?.title))) {
            actions.push(['Manage Site', 'admin', LINKS.Admin, null, SettingsIcon, 0]);
        }
        actions.push(['Shop', 'shop', LINKS.Shopping, null, ShopIcon, 0],
            ['Profile', 'profile', LINKS.Profile, null, PersonIcon, 0],
            ['Cart', 'cart', LINKS.Cart, null, ShoppingCartIcon, cart?.items?.length ?? 0]);
    }

    return actions;
}