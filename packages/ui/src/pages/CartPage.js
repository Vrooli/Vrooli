import React, { useState, useCallback, useEffect } from 'react';
import PropTypes from "prop-types";
import { useHistory } from 'react-router';
import { LINKS, PUBS, PubSub } from 'utils';
import { Button } from '@material-ui/core';
import { CartTable } from 'components';
import {
    ArrowBack as ArrowBackIcon,
    ArrowForward as ArrowForwardIcon,
    Update as UpdateIcon
} from '@material-ui/icons';
import { updateOrderMutation, submitOrderMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { Typography, Grid } from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import _ from 'lodash';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { pageStyles } from './styles';
import { combineStyles } from 'utils';

const componentStyles = (theme) => ({
    padTop: {
        paddingTop: theme.spacing(2),
    },
    gridItem: {
        display: 'flex',
    },
    optionsContainer: {
        width: 'fit-content',
        justifyContent: 'center',
        '& > *': {
            margin: theme.spacing(1),
        },
    },
})

const useStyles = makeStyles(combineStyles(pageStyles, componentStyles));

function CartPage({
    business,
    cart,
    onSessionUpdate
}) {
    let history = useHistory();
    const classes = useStyles();
    // Holds cart changes before update is final
    const [changedCart, setChangedCart] = useState(null);
    const [updateOrder, {loading}] = useMutation(updateOrderMutation);
    const [submitOrder] = useMutation(submitOrderMutation);

    useEffect(() => {
        setChangedCart(cart);
    }, [cart])

    const orderUpdate = () => {
        if (!cart?.customer?.id) {
            PubSub.publish(PUBS.Snack, {message: 'Failed to update order.', severity: 'error' });
            return;
        }
        mutationWrapper({
            mutation: updateOrder,
            data: { variables: { input: { 
                id: changedCart.id,
                desiredDeliveryDate: changedCart.desiredDeliveryDate,
                isDelivery: changedCart.isDelivery,
                specialInstructions: changedCart.specialInstructions,
                items: changedCart.items.map(i => ({ id: i.id, quantity: i.quantity }))} } },
            successCondition: (response) => response.data.updateOrder,
            onSuccess: () => onSessionUpdate(),
            successMessage: () => 'Order successfully updated.',
        })
    }

    const requestQuote = useCallback(() => {
        mutationWrapper({
            mutation: submitOrder,
            data: { variables: { id: cart.id } },
            onSuccess: () => {PubSub.publish(PUBS.AlertDialog, { message: 'Order submitted! We will be in touch with you soon😊' }); onSessionUpdate()},
            onError: () => PubSub.publish(PUBS.AlertDialog, {message: `Failed to submit order. Please contact ${business?.BUSINESS_NAME?.Short}`, severity: 'error'})
        })
    }, [submitOrder, cart, onSessionUpdate, business])

    const finalizeOrder = useCallback(() => {
        // Make sure order is updated
        if (!_.isEqual(cart, changedCart)) {
            PubSub.publish(PUBS.Snack, {message: 'Please click "UPDATE ORDER" before submitting.', severity: 'warning'});
            return;
        }
        // Disallow empty orders
        if (cart.items.length <= 0) {
            PubSub.publish(PUBS.Snack, {message: 'Cannot finalize order - cart is empty.', severity: 'warning'});
            return;
        }
        PubSub.publish(PUBS.AlertDialog, {
            message: `This will submit the order to ${business?.BUSINESS_NAME?.Short}. We will contact you for further information. Are you sure you want to continue?`,
            firstButtonText: 'Yes',
            firstButtonClicked: requestQuote,
            secondButtonText: 'No',
        });
    }, [changedCart, cart, requestQuote, business]);

    let options = (
        <Grid className={classes.padTop} container spacing={2}>
            <Grid className={classes.gridItem} justify="center" item xs={12} sm={4}>
                <Button 
                    fullWidth 
                    startIcon={<ArrowBackIcon />} 
                    onClick={() => history.push(LINKS.Shopping)}
                    disabled={loading || (changedCart !== null && !_.isEqual(cart, changedCart))}
                >Continue Shopping</Button>
            </Grid>
            <Grid className={classes.gridItem} justify="center" item xs={12} sm={4}>
                <Button 
                    fullWidth 
                    startIcon={<UpdateIcon />} 
                    onClick={orderUpdate}
                    disabled={loading || (changedCart === null || _.isEqual(cart, changedCart))}
                >Update Order</Button>
            </Grid>
            <Grid className={classes.gridItem} justify="center" item xs={12} sm={4}>
                <Button 
                    fullWidth 
                    endIcon={<ArrowForwardIcon />} 
                    onClick={finalizeOrder}
                    disabled={loading || changedCart === null || !_.isEqual(cart, changedCart)}
                >Request Quote</Button>
            </Grid>
        </Grid>
    )

    return (
        <div id='page'>
            <div className={classes.header}>
                <Typography variant="h3" component="h1">Cart</Typography>
            </div>
            { options}
            <CartTable className={classes.padTop} cart={changedCart} onUpdate={(d) => setChangedCart(d)} />
            { options}
        </div>
    );
}

CartPage.propTypes = {
    cart: PropTypes.object,
}

export { CartPage };
