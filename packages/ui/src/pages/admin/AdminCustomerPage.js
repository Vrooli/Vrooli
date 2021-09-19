import React, { useEffect, useState } from 'react';
import { customersQuery } from 'graphql/query';
import { useQuery } from '@apollo/client';
import { PUBS, PubSub } from 'utils';
import {
    AdminBreadcrumbs,
    CustomerCard
} from 'components';
import { makeStyles } from '@material-ui/styles';
import { Button, Typography } from '@material-ui/core';
import { useTheme } from '@emotion/react';
import { CustomerDialog } from 'components/dialogs/CustomerDialog';
import { NewCustomerDialog } from 'components/dialogs/NewCustomerDialog';
import { pageStyles } from '../styles';
import { combineStyles } from 'utils';

const componentStyles = (theme) => ({
    cardFlex: {
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(300px, .5fr))',
        gridGap: '20px',
    },
})

const useStyles = makeStyles(combineStyles(pageStyles, componentStyles));

function AdminCustomerPage() {
    const classes = useStyles();
    const theme = useTheme();
    const [customers, setCustomers] = useState(null);
    const [selectedCustomer, setSelectedCustomer] = useState(null);
    const [newCustomerOpen, setNewCustomerOpen] = useState(false);
    const { error, data } = useQuery(customersQuery, { pollInterval: 5000 });
    if (error) { 
        PubSub.publish(PUBS.Snack, { message: error.message, severity: 'error', data: error });
    }
    useEffect(() => {
        setCustomers(data?.customers);
    }, [data])

    return (
        <div id="page">
            <CustomerDialog
                customer={selectedCustomer}
                open={selectedCustomer !== null}
                onClose={() => setSelectedCustomer(null)} />
            <NewCustomerDialog
                open={newCustomerOpen}
                onClose={() => setNewCustomerOpen(false)} />
            <AdminBreadcrumbs textColor={theme.palette.secondary.dark} />
            <div className={classes.header}>
                <Typography variant="h3" component="h1">Manage Customers</Typography>
                <Button color="secondary" onClick={() => setNewCustomerOpen(true)}>Create Customer</Button>
            </div>
            <div className={classes.cardFlex}>
                {customers?.map((c, index) =>
                <CustomerCard 
                    key={index}
                    onEdit={setSelectedCustomer}
                    customer={c}
                />)}
            </div>
        </div >
    );
}

AdminCustomerPage.propTypes = {
}

export { AdminCustomerPage };