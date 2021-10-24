import { useCallback, useEffect, useMemo, useState } from 'react';
import { customersQuery } from 'graphql/query';
import { useQuery } from '@apollo/client';
import { PUBS } from 'utils';
import PubSub from 'pubsub-js';
import {
    AdminBreadcrumbs,
    CustomerCard
} from 'components';
import { makeStyles } from '@material-ui/styles';
import { useTheme } from '@material-ui/core';
import { Button, Typography } from '@material-ui/core';
import { CustomerDialog } from 'components/dialogs/CustomerDialog';
import { NewCustomerDialog } from 'components/dialogs/NewCustomerDialog';
import { pageStyles } from '../styles';
import { combineStyles } from 'utils';
import { customers, customers_customers } from 'graphql/generated/customers';

const componentStyles = () => ({
    cardFlex: {
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(300px, .5fr))',
        gridGap: '20px',
    },
})

const useStyles = makeStyles(combineStyles(pageStyles, componentStyles));

export const AdminCustomerPage = () => {
    const classes = useStyles();
    const theme = useTheme();
    const [customers, setCustomers] = useState<customers_customers[]>([]);
    const [selectedCustomer, setSelectedCustomer] = useState<customers_customers | null>(null);
    const [newCustomerOpen, setNewCustomerOpen] = useState(false);
    const { error, data } = useQuery<customers>(customersQuery, { pollInterval: 5000 });
    if (error) {
        PubSub.publish(PUBS.Snack, { message: error.message, severity: 'error', data: error });
    }
    useEffect(() => {
        setCustomers(data?.customers ?? []);
    }, [data])

    const clearSelected = useCallback(() => setSelectedCustomer(null), []);
    const openDialog = useCallback(() => setNewCustomerOpen(true), []);
    const closeDialog = useCallback(() => setNewCustomerOpen(false), []);

    const customerCards = useMemo(() => (
        customers?.map((c, index) =>
            <CustomerCard
                key={index}
                onEdit={(data) => setSelectedCustomer(data)}
                customer={c}
            />)
    ), [customers])

    return (
        <div id="page">
            <CustomerDialog
                customer={selectedCustomer}
                open={selectedCustomer !== null}
                onClose={clearSelected} />
            <NewCustomerDialog
                open={newCustomerOpen}
                onClose={closeDialog} />
            <AdminBreadcrumbs textColor={theme.palette.secondary.dark} />
            <div className={classes.header}>
                <Typography variant="h3" component="h1">Manage Customers</Typography>
                <Button color="secondary" onClick={openDialog}>Create Customer</Button>
            </div>
            <div className={classes.cardFlex}>
                {customerCards}
            </div>
        </div >
    );
}