export const auth_guestLogIn = {
  "fieldName": "guestLogIn",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "guestLogIn",
        "loc": {
          "start": 300,
          "end": 310
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLoggedIn",
              "loc": {
                "start": 317,
                "end": 327
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 317,
              "end": 327
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeZone",
              "loc": {
                "start": 332,
                "end": 340
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 332,
              "end": 340
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "users",
              "loc": {
                "start": 345,
                "end": 350
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "activeFocusMode",
                    "loc": {
                      "start": 361,
                      "end": 376
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "mode",
                          "loc": {
                            "start": 391,
                            "end": 395
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filters",
                                "loc": {
                                  "start": 414,
                                  "end": 421
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 444,
                                        "end": 446
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 444,
                                      "end": 446
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "filterType",
                                      "loc": {
                                        "start": 467,
                                        "end": 477
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 467,
                                      "end": 477
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 498,
                                        "end": 501
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 528,
                                              "end": 530
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 528,
                                            "end": 530
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 555,
                                              "end": 565
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 555,
                                            "end": 565
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "tag",
                                            "loc": {
                                              "start": 590,
                                              "end": 593
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 590,
                                            "end": 593
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bookmarks",
                                            "loc": {
                                              "start": 618,
                                              "end": 627
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 618,
                                            "end": 627
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 652,
                                              "end": 664
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 695,
                                                    "end": 697
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 695,
                                                  "end": 697
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 726,
                                                    "end": 734
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 726,
                                                  "end": 734
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 763,
                                                    "end": 774
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 763,
                                                  "end": 774
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 665,
                                              "end": 800
                                            }
                                          },
                                          "loc": {
                                            "start": 652,
                                            "end": 800
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 825,
                                              "end": 828
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isOwn",
                                                  "loc": {
                                                    "start": 859,
                                                    "end": 864
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 859,
                                                  "end": 864
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 893,
                                                    "end": 905
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 893,
                                                  "end": 905
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 829,
                                              "end": 931
                                            }
                                          },
                                          "loc": {
                                            "start": 825,
                                            "end": 931
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 502,
                                        "end": 953
                                      }
                                    },
                                    "loc": {
                                      "start": 498,
                                      "end": 953
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "focusMode",
                                      "loc": {
                                        "start": 974,
                                        "end": 983
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "labels",
                                            "loc": {
                                              "start": 1010,
                                              "end": 1016
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 1047,
                                                    "end": 1049
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1047,
                                                  "end": 1049
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "color",
                                                  "loc": {
                                                    "start": 1078,
                                                    "end": 1083
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1078,
                                                  "end": 1083
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "label",
                                                  "loc": {
                                                    "start": 1112,
                                                    "end": 1117
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1112,
                                                  "end": 1117
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1017,
                                              "end": 1143
                                            }
                                          },
                                          "loc": {
                                            "start": 1010,
                                            "end": 1143
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "schedule",
                                            "loc": {
                                              "start": 1168,
                                              "end": 1176
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "FragmentSpread",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "Schedule_common",
                                                  "loc": {
                                                    "start": 1210,
                                                    "end": 1225
                                                  }
                                                },
                                                "directives": [],
                                                "loc": {
                                                  "start": 1207,
                                                  "end": 1225
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1177,
                                              "end": 1251
                                            }
                                          },
                                          "loc": {
                                            "start": 1168,
                                            "end": 1251
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 1276,
                                              "end": 1278
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1276,
                                            "end": 1278
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 1303,
                                              "end": 1307
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1303,
                                            "end": 1307
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 1332,
                                              "end": 1343
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1332,
                                            "end": 1343
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 984,
                                        "end": 1365
                                      }
                                    },
                                    "loc": {
                                      "start": 974,
                                      "end": 1365
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 422,
                                  "end": 1383
                                }
                              },
                              "loc": {
                                "start": 414,
                                "end": 1383
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "labels",
                                "loc": {
                                  "start": 1400,
                                  "end": 1406
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 1429,
                                        "end": 1431
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1429,
                                      "end": 1431
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "color",
                                      "loc": {
                                        "start": 1452,
                                        "end": 1457
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1452,
                                      "end": 1457
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "label",
                                      "loc": {
                                        "start": 1478,
                                        "end": 1483
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1478,
                                      "end": 1483
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 1407,
                                  "end": 1501
                                }
                              },
                              "loc": {
                                "start": 1400,
                                "end": 1501
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderList",
                                "loc": {
                                  "start": 1518,
                                  "end": 1530
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 1553,
                                        "end": 1555
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1553,
                                      "end": 1555
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 1576,
                                        "end": 1586
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1576,
                                      "end": 1586
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 1607,
                                        "end": 1617
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1607,
                                      "end": 1617
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminders",
                                      "loc": {
                                        "start": 1638,
                                        "end": 1647
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 1674,
                                              "end": 1676
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1674,
                                            "end": 1676
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 1701,
                                              "end": 1711
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1701,
                                            "end": 1711
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 1736,
                                              "end": 1746
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1736,
                                            "end": 1746
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 1771,
                                              "end": 1775
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1771,
                                            "end": 1775
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 1800,
                                              "end": 1811
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1800,
                                            "end": 1811
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 1836,
                                              "end": 1843
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1836,
                                            "end": 1843
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 1868,
                                              "end": 1873
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1868,
                                            "end": 1873
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 1898,
                                              "end": 1908
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1898,
                                            "end": 1908
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderItems",
                                            "loc": {
                                              "start": 1933,
                                              "end": 1946
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 1977,
                                                    "end": 1979
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1977,
                                                  "end": 1979
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 2008,
                                                    "end": 2018
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2008,
                                                  "end": 2018
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 2047,
                                                    "end": 2057
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2047,
                                                  "end": 2057
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 2086,
                                                    "end": 2090
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2086,
                                                  "end": 2090
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 2119,
                                                    "end": 2130
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2119,
                                                  "end": 2130
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 2159,
                                                    "end": 2166
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2159,
                                                  "end": 2166
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 2195,
                                                    "end": 2200
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2195,
                                                  "end": 2200
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 2229,
                                                    "end": 2239
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2229,
                                                  "end": 2239
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1947,
                                              "end": 2265
                                            }
                                          },
                                          "loc": {
                                            "start": 1933,
                                            "end": 2265
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1648,
                                        "end": 2287
                                      }
                                    },
                                    "loc": {
                                      "start": 1638,
                                      "end": 2287
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 1531,
                                  "end": 2305
                                }
                              },
                              "loc": {
                                "start": 1518,
                                "end": 2305
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resourceList",
                                "loc": {
                                  "start": 2322,
                                  "end": 2334
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 2357,
                                        "end": 2359
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2357,
                                      "end": 2359
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 2380,
                                        "end": 2390
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2380,
                                      "end": 2390
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 2411,
                                        "end": 2423
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 2450,
                                              "end": 2452
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2450,
                                            "end": 2452
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 2477,
                                              "end": 2485
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2477,
                                            "end": 2485
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 2510,
                                              "end": 2521
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2510,
                                            "end": 2521
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 2546,
                                              "end": 2550
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2546,
                                            "end": 2550
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 2424,
                                        "end": 2572
                                      }
                                    },
                                    "loc": {
                                      "start": 2411,
                                      "end": 2572
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resources",
                                      "loc": {
                                        "start": 2593,
                                        "end": 2602
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 2629,
                                              "end": 2631
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2629,
                                            "end": 2631
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 2656,
                                              "end": 2661
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2656,
                                            "end": 2661
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "link",
                                            "loc": {
                                              "start": 2686,
                                              "end": 2690
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2686,
                                            "end": 2690
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "usedFor",
                                            "loc": {
                                              "start": 2715,
                                              "end": 2722
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2715,
                                            "end": 2722
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 2747,
                                              "end": 2759
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 2790,
                                                    "end": 2792
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2790,
                                                  "end": 2792
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 2821,
                                                    "end": 2829
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2821,
                                                  "end": 2829
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 2858,
                                                    "end": 2869
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2858,
                                                  "end": 2869
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 2898,
                                                    "end": 2902
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2898,
                                                  "end": 2902
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2760,
                                              "end": 2928
                                            }
                                          },
                                          "loc": {
                                            "start": 2747,
                                            "end": 2928
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 2603,
                                        "end": 2950
                                      }
                                    },
                                    "loc": {
                                      "start": 2593,
                                      "end": 2950
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2335,
                                  "end": 2968
                                }
                              },
                              "loc": {
                                "start": 2322,
                                "end": 2968
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "schedule",
                                "loc": {
                                  "start": 2985,
                                  "end": 2993
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "FragmentSpread",
                                    "name": {
                                      "kind": "Name",
                                      "value": "Schedule_common",
                                      "loc": {
                                        "start": 3019,
                                        "end": 3034
                                      }
                                    },
                                    "directives": [],
                                    "loc": {
                                      "start": 3016,
                                      "end": 3034
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2994,
                                  "end": 3052
                                }
                              },
                              "loc": {
                                "start": 2985,
                                "end": 3052
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3069,
                                  "end": 3071
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3069,
                                "end": 3071
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 3088,
                                  "end": 3092
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3088,
                                "end": 3092
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 3109,
                                  "end": 3120
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3109,
                                "end": 3120
                              }
                            }
                          ],
                          "loc": {
                            "start": 396,
                            "end": 3134
                          }
                        },
                        "loc": {
                          "start": 391,
                          "end": 3134
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopCondition",
                          "loc": {
                            "start": 3147,
                            "end": 3160
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3147,
                          "end": 3160
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopTime",
                          "loc": {
                            "start": 3173,
                            "end": 3181
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3173,
                          "end": 3181
                        }
                      }
                    ],
                    "loc": {
                      "start": 377,
                      "end": 3191
                    }
                  },
                  "loc": {
                    "start": 361,
                    "end": 3191
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apisCount",
                    "loc": {
                      "start": 3200,
                      "end": 3209
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3200,
                    "end": 3209
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bookmarkLists",
                    "loc": {
                      "start": 3218,
                      "end": 3231
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 3246,
                            "end": 3248
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3246,
                          "end": 3248
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 3261,
                            "end": 3271
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3261,
                          "end": 3271
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 3284,
                            "end": 3294
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3284,
                          "end": 3294
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 3307,
                            "end": 3312
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3307,
                          "end": 3312
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bookmarksCount",
                          "loc": {
                            "start": 3325,
                            "end": 3339
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3325,
                          "end": 3339
                        }
                      }
                    ],
                    "loc": {
                      "start": 3232,
                      "end": 3349
                    }
                  },
                  "loc": {
                    "start": 3218,
                    "end": 3349
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusModes",
                    "loc": {
                      "start": 3358,
                      "end": 3368
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "filters",
                          "loc": {
                            "start": 3383,
                            "end": 3390
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3409,
                                  "end": 3411
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3409,
                                "end": 3411
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filterType",
                                "loc": {
                                  "start": 3428,
                                  "end": 3438
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3428,
                                "end": 3438
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "tag",
                                "loc": {
                                  "start": 3455,
                                  "end": 3458
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3481,
                                        "end": 3483
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3481,
                                      "end": 3483
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3504,
                                        "end": 3514
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3504,
                                      "end": 3514
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 3535,
                                        "end": 3538
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3535,
                                      "end": 3538
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bookmarks",
                                      "loc": {
                                        "start": 3559,
                                        "end": 3568
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3559,
                                      "end": 3568
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 3589,
                                        "end": 3601
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3628,
                                              "end": 3630
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3628,
                                            "end": 3630
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 3655,
                                              "end": 3663
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3655,
                                            "end": 3663
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3688,
                                              "end": 3699
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3688,
                                            "end": 3699
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3602,
                                        "end": 3721
                                      }
                                    },
                                    "loc": {
                                      "start": 3589,
                                      "end": 3721
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 3742,
                                        "end": 3745
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isOwn",
                                            "loc": {
                                              "start": 3772,
                                              "end": 3777
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3772,
                                            "end": 3777
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 3802,
                                              "end": 3814
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3802,
                                            "end": 3814
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3746,
                                        "end": 3836
                                      }
                                    },
                                    "loc": {
                                      "start": 3742,
                                      "end": 3836
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3459,
                                  "end": 3854
                                }
                              },
                              "loc": {
                                "start": 3455,
                                "end": 3854
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "focusMode",
                                "loc": {
                                  "start": 3871,
                                  "end": 3880
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "labels",
                                      "loc": {
                                        "start": 3903,
                                        "end": 3909
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3936,
                                              "end": 3938
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3936,
                                            "end": 3938
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "color",
                                            "loc": {
                                              "start": 3963,
                                              "end": 3968
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3963,
                                            "end": 3968
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "label",
                                            "loc": {
                                              "start": 3993,
                                              "end": 3998
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3993,
                                            "end": 3998
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3910,
                                        "end": 4020
                                      }
                                    },
                                    "loc": {
                                      "start": 3903,
                                      "end": 4020
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "schedule",
                                      "loc": {
                                        "start": 4041,
                                        "end": 4049
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "FragmentSpread",
                                          "name": {
                                            "kind": "Name",
                                            "value": "Schedule_common",
                                            "loc": {
                                              "start": 4079,
                                              "end": 4094
                                            }
                                          },
                                          "directives": [],
                                          "loc": {
                                            "start": 4076,
                                            "end": 4094
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4050,
                                        "end": 4116
                                      }
                                    },
                                    "loc": {
                                      "start": 4041,
                                      "end": 4116
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4137,
                                        "end": 4139
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4137,
                                      "end": 4139
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 4160,
                                        "end": 4164
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4160,
                                      "end": 4164
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 4185,
                                        "end": 4196
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4185,
                                      "end": 4196
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3881,
                                  "end": 4214
                                }
                              },
                              "loc": {
                                "start": 3871,
                                "end": 4214
                              }
                            }
                          ],
                          "loc": {
                            "start": 3391,
                            "end": 4228
                          }
                        },
                        "loc": {
                          "start": 3383,
                          "end": 4228
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 4241,
                            "end": 4247
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4266,
                                  "end": 4268
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4266,
                                "end": 4268
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 4285,
                                  "end": 4290
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4285,
                                "end": 4290
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 4307,
                                  "end": 4312
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4307,
                                "end": 4312
                              }
                            }
                          ],
                          "loc": {
                            "start": 4248,
                            "end": 4326
                          }
                        },
                        "loc": {
                          "start": 4241,
                          "end": 4326
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderList",
                          "loc": {
                            "start": 4339,
                            "end": 4351
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4370,
                                  "end": 4372
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4370,
                                "end": 4372
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 4389,
                                  "end": 4399
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4389,
                                "end": 4399
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 4416,
                                  "end": 4426
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4416,
                                "end": 4426
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminders",
                                "loc": {
                                  "start": 4443,
                                  "end": 4452
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4475,
                                        "end": 4477
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4475,
                                      "end": 4477
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4498,
                                        "end": 4508
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4498,
                                      "end": 4508
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 4529,
                                        "end": 4539
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4529,
                                      "end": 4539
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 4560,
                                        "end": 4564
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4560,
                                      "end": 4564
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 4585,
                                        "end": 4596
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4585,
                                      "end": 4596
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 4617,
                                        "end": 4624
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4617,
                                      "end": 4624
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 4645,
                                        "end": 4650
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4645,
                                      "end": 4650
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 4671,
                                        "end": 4681
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4671,
                                      "end": 4681
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderItems",
                                      "loc": {
                                        "start": 4702,
                                        "end": 4715
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 4742,
                                              "end": 4744
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4742,
                                            "end": 4744
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 4769,
                                              "end": 4779
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4769,
                                            "end": 4779
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 4804,
                                              "end": 4814
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4804,
                                            "end": 4814
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 4839,
                                              "end": 4843
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4839,
                                            "end": 4843
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4868,
                                              "end": 4879
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4868,
                                            "end": 4879
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 4904,
                                              "end": 4911
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4904,
                                            "end": 4911
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 4936,
                                              "end": 4941
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4936,
                                            "end": 4941
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 4966,
                                              "end": 4976
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4966,
                                            "end": 4976
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4716,
                                        "end": 4998
                                      }
                                    },
                                    "loc": {
                                      "start": 4702,
                                      "end": 4998
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4453,
                                  "end": 5016
                                }
                              },
                              "loc": {
                                "start": 4443,
                                "end": 5016
                              }
                            }
                          ],
                          "loc": {
                            "start": 4352,
                            "end": 5030
                          }
                        },
                        "loc": {
                          "start": 4339,
                          "end": 5030
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resourceList",
                          "loc": {
                            "start": 5043,
                            "end": 5055
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 5074,
                                  "end": 5076
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5074,
                                "end": 5076
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 5093,
                                  "end": 5103
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5093,
                                "end": 5103
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 5120,
                                  "end": 5132
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 5155,
                                        "end": 5157
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5155,
                                      "end": 5157
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 5178,
                                        "end": 5186
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5178,
                                      "end": 5186
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 5207,
                                        "end": 5218
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5207,
                                      "end": 5218
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 5239,
                                        "end": 5243
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5239,
                                      "end": 5243
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5133,
                                  "end": 5261
                                }
                              },
                              "loc": {
                                "start": 5120,
                                "end": 5261
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resources",
                                "loc": {
                                  "start": 5278,
                                  "end": 5287
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 5310,
                                        "end": 5312
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5310,
                                      "end": 5312
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 5333,
                                        "end": 5338
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5333,
                                      "end": 5338
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "link",
                                      "loc": {
                                        "start": 5359,
                                        "end": 5363
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5359,
                                      "end": 5363
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "usedFor",
                                      "loc": {
                                        "start": 5384,
                                        "end": 5391
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5384,
                                      "end": 5391
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 5412,
                                        "end": 5424
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 5451,
                                              "end": 5453
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5451,
                                            "end": 5453
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 5478,
                                              "end": 5486
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5478,
                                            "end": 5486
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 5511,
                                              "end": 5522
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5511,
                                            "end": 5522
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 5547,
                                              "end": 5551
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5547,
                                            "end": 5551
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5425,
                                        "end": 5573
                                      }
                                    },
                                    "loc": {
                                      "start": 5412,
                                      "end": 5573
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5288,
                                  "end": 5591
                                }
                              },
                              "loc": {
                                "start": 5278,
                                "end": 5591
                              }
                            }
                          ],
                          "loc": {
                            "start": 5056,
                            "end": 5605
                          }
                        },
                        "loc": {
                          "start": 5043,
                          "end": 5605
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 5618,
                            "end": 5626
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Schedule_common",
                                "loc": {
                                  "start": 5648,
                                  "end": 5663
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 5645,
                                "end": 5663
                              }
                            }
                          ],
                          "loc": {
                            "start": 5627,
                            "end": 5677
                          }
                        },
                        "loc": {
                          "start": 5618,
                          "end": 5677
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5690,
                            "end": 5692
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5690,
                          "end": 5692
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5705,
                            "end": 5709
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5705,
                          "end": 5709
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5722,
                            "end": 5733
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5722,
                          "end": 5733
                        }
                      }
                    ],
                    "loc": {
                      "start": 3369,
                      "end": 5743
                    }
                  },
                  "loc": {
                    "start": 3358,
                    "end": 5743
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 5752,
                      "end": 5758
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5752,
                    "end": 5758
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasPremium",
                    "loc": {
                      "start": 5767,
                      "end": 5777
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5767,
                    "end": 5777
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5786,
                      "end": 5788
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5786,
                    "end": 5788
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "languages",
                    "loc": {
                      "start": 5797,
                      "end": 5806
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5797,
                    "end": 5806
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membershipsCount",
                    "loc": {
                      "start": 5815,
                      "end": 5831
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5815,
                    "end": 5831
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5840,
                      "end": 5844
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5840,
                    "end": 5844
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "notesCount",
                    "loc": {
                      "start": 5853,
                      "end": 5863
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5853,
                    "end": 5863
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "projectsCount",
                    "loc": {
                      "start": 5872,
                      "end": 5885
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5872,
                    "end": 5885
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "questionsAskedCount",
                    "loc": {
                      "start": 5894,
                      "end": 5913
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5894,
                    "end": 5913
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routinesCount",
                    "loc": {
                      "start": 5922,
                      "end": 5935
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5922,
                    "end": 5935
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractsCount",
                    "loc": {
                      "start": 5944,
                      "end": 5963
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5944,
                    "end": 5963
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardsCount",
                    "loc": {
                      "start": 5972,
                      "end": 5986
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5972,
                    "end": 5986
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "theme",
                    "loc": {
                      "start": 5995,
                      "end": 6000
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5995,
                    "end": 6000
                  }
                }
              ],
              "loc": {
                "start": 351,
                "end": 6006
              }
            },
            "loc": {
              "start": 345,
              "end": 6006
            }
          }
        ],
        "loc": {
          "start": 311,
          "end": 6010
        }
      },
      "loc": {
        "start": 300,
        "end": 6010
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Schedule_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_common",
        "loc": {
          "start": 9,
          "end": 24
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 28,
            "end": 36
          }
        },
        "loc": {
          "start": 28,
          "end": 36
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 39,
                "end": 41
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 39,
              "end": 41
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 42,
                "end": 52
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 42,
              "end": 52
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 53,
                "end": 63
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 53,
              "end": 63
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 64,
                "end": 73
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 64,
              "end": 73
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 74,
                "end": 81
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 74,
              "end": 81
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 82,
                "end": 90
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 82,
              "end": 90
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 91,
                "end": 101
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 108,
                      "end": 110
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 108,
                    "end": 110
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 115,
                      "end": 132
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 115,
                    "end": 132
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 137,
                      "end": 149
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 137,
                    "end": 149
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 154,
                      "end": 164
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 154,
                    "end": 164
                  }
                }
              ],
              "loc": {
                "start": 102,
                "end": 166
              }
            },
            "loc": {
              "start": 91,
              "end": 166
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 167,
                "end": 178
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 185,
                      "end": 187
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 185,
                    "end": 187
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 192,
                      "end": 206
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 192,
                    "end": 206
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 211,
                      "end": 219
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 211,
                    "end": 219
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 224,
                      "end": 233
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 224,
                    "end": 233
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 238,
                      "end": 248
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 238,
                    "end": 248
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 253,
                      "end": 258
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 253,
                    "end": 258
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 263,
                      "end": 270
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 263,
                    "end": 270
                  }
                }
              ],
              "loc": {
                "start": 179,
                "end": 272
              }
            },
            "loc": {
              "start": 167,
              "end": 272
            }
          }
        ],
        "loc": {
          "start": 37,
          "end": 274
        }
      },
      "loc": {
        "start": 0,
        "end": 274
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "mutation",
    "name": {
      "kind": "Name",
      "value": "guestLogIn",
      "loc": {
        "start": 285,
        "end": 295
      }
    },
    "variableDefinitions": [],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "guestLogIn",
            "loc": {
              "start": 300,
              "end": 310
            }
          },
          "arguments": [],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "isLoggedIn",
                  "loc": {
                    "start": 317,
                    "end": 327
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 317,
                  "end": 327
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "timeZone",
                  "loc": {
                    "start": 332,
                    "end": 340
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 332,
                  "end": 340
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "users",
                  "loc": {
                    "start": 345,
                    "end": 350
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "activeFocusMode",
                        "loc": {
                          "start": 361,
                          "end": 376
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "mode",
                              "loc": {
                                "start": 391,
                                "end": 395
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "filters",
                                    "loc": {
                                      "start": 414,
                                      "end": 421
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 444,
                                            "end": 446
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 444,
                                          "end": 446
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "filterType",
                                          "loc": {
                                            "start": 467,
                                            "end": 477
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 467,
                                          "end": 477
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "tag",
                                          "loc": {
                                            "start": 498,
                                            "end": 501
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 528,
                                                  "end": 530
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 528,
                                                "end": 530
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 555,
                                                  "end": 565
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 555,
                                                "end": 565
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "tag",
                                                "loc": {
                                                  "start": 590,
                                                  "end": 593
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 590,
                                                "end": 593
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "bookmarks",
                                                "loc": {
                                                  "start": 618,
                                                  "end": 627
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 618,
                                                "end": 627
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 652,
                                                  "end": 664
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 695,
                                                        "end": 697
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 695,
                                                      "end": 697
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 726,
                                                        "end": 734
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 726,
                                                      "end": 734
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 763,
                                                        "end": 774
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 763,
                                                      "end": 774
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 665,
                                                  "end": 800
                                                }
                                              },
                                              "loc": {
                                                "start": 652,
                                                "end": 800
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "you",
                                                "loc": {
                                                  "start": 825,
                                                  "end": 828
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isOwn",
                                                      "loc": {
                                                        "start": 859,
                                                        "end": 864
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 859,
                                                      "end": 864
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isBookmarked",
                                                      "loc": {
                                                        "start": 893,
                                                        "end": 905
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 893,
                                                      "end": 905
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 829,
                                                  "end": 931
                                                }
                                              },
                                              "loc": {
                                                "start": 825,
                                                "end": 931
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 502,
                                            "end": 953
                                          }
                                        },
                                        "loc": {
                                          "start": 498,
                                          "end": 953
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "focusMode",
                                          "loc": {
                                            "start": 974,
                                            "end": 983
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "labels",
                                                "loc": {
                                                  "start": 1010,
                                                  "end": 1016
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 1047,
                                                        "end": 1049
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1047,
                                                      "end": 1049
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "color",
                                                      "loc": {
                                                        "start": 1078,
                                                        "end": 1083
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1078,
                                                      "end": 1083
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "label",
                                                      "loc": {
                                                        "start": 1112,
                                                        "end": 1117
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1112,
                                                      "end": 1117
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1017,
                                                  "end": 1143
                                                }
                                              },
                                              "loc": {
                                                "start": 1010,
                                                "end": 1143
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "schedule",
                                                "loc": {
                                                  "start": 1168,
                                                  "end": 1176
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "FragmentSpread",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "Schedule_common",
                                                      "loc": {
                                                        "start": 1210,
                                                        "end": 1225
                                                      }
                                                    },
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1207,
                                                      "end": 1225
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1177,
                                                  "end": 1251
                                                }
                                              },
                                              "loc": {
                                                "start": 1168,
                                                "end": 1251
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 1276,
                                                  "end": 1278
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1276,
                                                "end": 1278
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 1303,
                                                  "end": 1307
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1303,
                                                "end": 1307
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 1332,
                                                  "end": 1343
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1332,
                                                "end": 1343
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 984,
                                            "end": 1365
                                          }
                                        },
                                        "loc": {
                                          "start": 974,
                                          "end": 1365
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 422,
                                      "end": 1383
                                    }
                                  },
                                  "loc": {
                                    "start": 414,
                                    "end": 1383
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "labels",
                                    "loc": {
                                      "start": 1400,
                                      "end": 1406
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 1429,
                                            "end": 1431
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1429,
                                          "end": 1431
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "color",
                                          "loc": {
                                            "start": 1452,
                                            "end": 1457
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1452,
                                          "end": 1457
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "label",
                                          "loc": {
                                            "start": 1478,
                                            "end": 1483
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1478,
                                          "end": 1483
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 1407,
                                      "end": 1501
                                    }
                                  },
                                  "loc": {
                                    "start": 1400,
                                    "end": 1501
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "reminderList",
                                    "loc": {
                                      "start": 1518,
                                      "end": 1530
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 1553,
                                            "end": 1555
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1553,
                                          "end": 1555
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 1576,
                                            "end": 1586
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1576,
                                          "end": 1586
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 1607,
                                            "end": 1617
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1607,
                                          "end": 1617
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminders",
                                          "loc": {
                                            "start": 1638,
                                            "end": 1647
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 1674,
                                                  "end": 1676
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1674,
                                                "end": 1676
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 1701,
                                                  "end": 1711
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1701,
                                                "end": 1711
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 1736,
                                                  "end": 1746
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1736,
                                                "end": 1746
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 1771,
                                                  "end": 1775
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1771,
                                                "end": 1775
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 1800,
                                                  "end": 1811
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1800,
                                                "end": 1811
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "dueDate",
                                                "loc": {
                                                  "start": 1836,
                                                  "end": 1843
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1836,
                                                "end": 1843
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 1868,
                                                  "end": 1873
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1868,
                                                "end": 1873
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isComplete",
                                                "loc": {
                                                  "start": 1898,
                                                  "end": 1908
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1898,
                                                "end": 1908
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "reminderItems",
                                                "loc": {
                                                  "start": 1933,
                                                  "end": 1946
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 1977,
                                                        "end": 1979
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1977,
                                                      "end": 1979
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 2008,
                                                        "end": 2018
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2008,
                                                      "end": 2018
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 2047,
                                                        "end": 2057
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2047,
                                                      "end": 2057
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 2086,
                                                        "end": 2090
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2086,
                                                      "end": 2090
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 2119,
                                                        "end": 2130
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2119,
                                                      "end": 2130
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "dueDate",
                                                      "loc": {
                                                        "start": 2159,
                                                        "end": 2166
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2159,
                                                      "end": 2166
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "index",
                                                      "loc": {
                                                        "start": 2195,
                                                        "end": 2200
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2195,
                                                      "end": 2200
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isComplete",
                                                      "loc": {
                                                        "start": 2229,
                                                        "end": 2239
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2229,
                                                      "end": 2239
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1947,
                                                  "end": 2265
                                                }
                                              },
                                              "loc": {
                                                "start": 1933,
                                                "end": 2265
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 1648,
                                            "end": 2287
                                          }
                                        },
                                        "loc": {
                                          "start": 1638,
                                          "end": 2287
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 1531,
                                      "end": 2305
                                    }
                                  },
                                  "loc": {
                                    "start": 1518,
                                    "end": 2305
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "resourceList",
                                    "loc": {
                                      "start": 2322,
                                      "end": 2334
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 2357,
                                            "end": 2359
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2357,
                                          "end": 2359
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 2380,
                                            "end": 2390
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2380,
                                          "end": 2390
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 2411,
                                            "end": 2423
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 2450,
                                                  "end": 2452
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2450,
                                                "end": 2452
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 2477,
                                                  "end": 2485
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2477,
                                                "end": 2485
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 2510,
                                                  "end": 2521
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2510,
                                                "end": 2521
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 2546,
                                                  "end": 2550
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2546,
                                                "end": 2550
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 2424,
                                            "end": 2572
                                          }
                                        },
                                        "loc": {
                                          "start": 2411,
                                          "end": 2572
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "resources",
                                          "loc": {
                                            "start": 2593,
                                            "end": 2602
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 2629,
                                                  "end": 2631
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2629,
                                                "end": 2631
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 2656,
                                                  "end": 2661
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2656,
                                                "end": 2661
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "link",
                                                "loc": {
                                                  "start": 2686,
                                                  "end": 2690
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2686,
                                                "end": 2690
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "usedFor",
                                                "loc": {
                                                  "start": 2715,
                                                  "end": 2722
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2715,
                                                "end": 2722
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 2747,
                                                  "end": 2759
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 2790,
                                                        "end": 2792
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2790,
                                                      "end": 2792
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 2821,
                                                        "end": 2829
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2821,
                                                      "end": 2829
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 2858,
                                                        "end": 2869
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2858,
                                                      "end": 2869
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 2898,
                                                        "end": 2902
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2898,
                                                      "end": 2902
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 2760,
                                                  "end": 2928
                                                }
                                              },
                                              "loc": {
                                                "start": 2747,
                                                "end": 2928
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 2603,
                                            "end": 2950
                                          }
                                        },
                                        "loc": {
                                          "start": 2593,
                                          "end": 2950
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 2335,
                                      "end": 2968
                                    }
                                  },
                                  "loc": {
                                    "start": 2322,
                                    "end": 2968
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "schedule",
                                    "loc": {
                                      "start": 2985,
                                      "end": 2993
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "FragmentSpread",
                                        "name": {
                                          "kind": "Name",
                                          "value": "Schedule_common",
                                          "loc": {
                                            "start": 3019,
                                            "end": 3034
                                          }
                                        },
                                        "directives": [],
                                        "loc": {
                                          "start": 3016,
                                          "end": 3034
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 2994,
                                      "end": 3052
                                    }
                                  },
                                  "loc": {
                                    "start": 2985,
                                    "end": 3052
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 3069,
                                      "end": 3071
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 3069,
                                    "end": 3071
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "name",
                                    "loc": {
                                      "start": 3088,
                                      "end": 3092
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 3088,
                                    "end": 3092
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "description",
                                    "loc": {
                                      "start": 3109,
                                      "end": 3120
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 3109,
                                    "end": 3120
                                  }
                                }
                              ],
                              "loc": {
                                "start": 396,
                                "end": 3134
                              }
                            },
                            "loc": {
                              "start": 391,
                              "end": 3134
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "stopCondition",
                              "loc": {
                                "start": 3147,
                                "end": 3160
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3147,
                              "end": 3160
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "stopTime",
                              "loc": {
                                "start": 3173,
                                "end": 3181
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3173,
                              "end": 3181
                            }
                          }
                        ],
                        "loc": {
                          "start": 377,
                          "end": 3191
                        }
                      },
                      "loc": {
                        "start": 361,
                        "end": 3191
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "apisCount",
                        "loc": {
                          "start": 3200,
                          "end": 3209
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 3200,
                        "end": 3209
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "bookmarkLists",
                        "loc": {
                          "start": 3218,
                          "end": 3231
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "id",
                              "loc": {
                                "start": 3246,
                                "end": 3248
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3246,
                              "end": 3248
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "created_at",
                              "loc": {
                                "start": 3261,
                                "end": 3271
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3261,
                              "end": 3271
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "updated_at",
                              "loc": {
                                "start": 3284,
                                "end": 3294
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3284,
                              "end": 3294
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "label",
                              "loc": {
                                "start": 3307,
                                "end": 3312
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3307,
                              "end": 3312
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "bookmarksCount",
                              "loc": {
                                "start": 3325,
                                "end": 3339
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3325,
                              "end": 3339
                            }
                          }
                        ],
                        "loc": {
                          "start": 3232,
                          "end": 3349
                        }
                      },
                      "loc": {
                        "start": 3218,
                        "end": 3349
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "focusModes",
                        "loc": {
                          "start": 3358,
                          "end": 3368
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "filters",
                              "loc": {
                                "start": 3383,
                                "end": 3390
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 3409,
                                      "end": 3411
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 3409,
                                    "end": 3411
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "filterType",
                                    "loc": {
                                      "start": 3428,
                                      "end": 3438
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 3428,
                                    "end": 3438
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "tag",
                                    "loc": {
                                      "start": 3455,
                                      "end": 3458
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 3481,
                                            "end": 3483
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3481,
                                          "end": 3483
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 3504,
                                            "end": 3514
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3504,
                                          "end": 3514
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "tag",
                                          "loc": {
                                            "start": 3535,
                                            "end": 3538
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3535,
                                          "end": 3538
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "bookmarks",
                                          "loc": {
                                            "start": 3559,
                                            "end": 3568
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3559,
                                          "end": 3568
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 3589,
                                            "end": 3601
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 3628,
                                                  "end": 3630
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3628,
                                                "end": 3630
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 3655,
                                                  "end": 3663
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3655,
                                                "end": 3663
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 3688,
                                                  "end": 3699
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3688,
                                                "end": 3699
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 3602,
                                            "end": 3721
                                          }
                                        },
                                        "loc": {
                                          "start": 3589,
                                          "end": 3721
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "you",
                                          "loc": {
                                            "start": 3742,
                                            "end": 3745
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isOwn",
                                                "loc": {
                                                  "start": 3772,
                                                  "end": 3777
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3772,
                                                "end": 3777
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isBookmarked",
                                                "loc": {
                                                  "start": 3802,
                                                  "end": 3814
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3802,
                                                "end": 3814
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 3746,
                                            "end": 3836
                                          }
                                        },
                                        "loc": {
                                          "start": 3742,
                                          "end": 3836
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3459,
                                      "end": 3854
                                    }
                                  },
                                  "loc": {
                                    "start": 3455,
                                    "end": 3854
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "focusMode",
                                    "loc": {
                                      "start": 3871,
                                      "end": 3880
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "labels",
                                          "loc": {
                                            "start": 3903,
                                            "end": 3909
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 3936,
                                                  "end": 3938
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3936,
                                                "end": 3938
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "color",
                                                "loc": {
                                                  "start": 3963,
                                                  "end": 3968
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3963,
                                                "end": 3968
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "label",
                                                "loc": {
                                                  "start": 3993,
                                                  "end": 3998
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3993,
                                                "end": 3998
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 3910,
                                            "end": 4020
                                          }
                                        },
                                        "loc": {
                                          "start": 3903,
                                          "end": 4020
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "schedule",
                                          "loc": {
                                            "start": 4041,
                                            "end": 4049
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "FragmentSpread",
                                              "name": {
                                                "kind": "Name",
                                                "value": "Schedule_common",
                                                "loc": {
                                                  "start": 4079,
                                                  "end": 4094
                                                }
                                              },
                                              "directives": [],
                                              "loc": {
                                                "start": 4076,
                                                "end": 4094
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 4050,
                                            "end": 4116
                                          }
                                        },
                                        "loc": {
                                          "start": 4041,
                                          "end": 4116
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 4137,
                                            "end": 4139
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4137,
                                          "end": 4139
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 4160,
                                            "end": 4164
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4160,
                                          "end": 4164
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 4185,
                                            "end": 4196
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4185,
                                          "end": 4196
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3881,
                                      "end": 4214
                                    }
                                  },
                                  "loc": {
                                    "start": 3871,
                                    "end": 4214
                                  }
                                }
                              ],
                              "loc": {
                                "start": 3391,
                                "end": 4228
                              }
                            },
                            "loc": {
                              "start": 3383,
                              "end": 4228
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "labels",
                              "loc": {
                                "start": 4241,
                                "end": 4247
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 4266,
                                      "end": 4268
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4266,
                                    "end": 4268
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "color",
                                    "loc": {
                                      "start": 4285,
                                      "end": 4290
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4285,
                                    "end": 4290
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "label",
                                    "loc": {
                                      "start": 4307,
                                      "end": 4312
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4307,
                                    "end": 4312
                                  }
                                }
                              ],
                              "loc": {
                                "start": 4248,
                                "end": 4326
                              }
                            },
                            "loc": {
                              "start": 4241,
                              "end": 4326
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "reminderList",
                              "loc": {
                                "start": 4339,
                                "end": 4351
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 4370,
                                      "end": 4372
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4370,
                                    "end": 4372
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 4389,
                                      "end": 4399
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4389,
                                    "end": 4399
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "updated_at",
                                    "loc": {
                                      "start": 4416,
                                      "end": 4426
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4416,
                                    "end": 4426
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "reminders",
                                    "loc": {
                                      "start": 4443,
                                      "end": 4452
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 4475,
                                            "end": 4477
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4475,
                                          "end": 4477
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 4498,
                                            "end": 4508
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4498,
                                          "end": 4508
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 4529,
                                            "end": 4539
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4529,
                                          "end": 4539
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 4560,
                                            "end": 4564
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4560,
                                          "end": 4564
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 4585,
                                            "end": 4596
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4585,
                                          "end": 4596
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "dueDate",
                                          "loc": {
                                            "start": 4617,
                                            "end": 4624
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4617,
                                          "end": 4624
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "index",
                                          "loc": {
                                            "start": 4645,
                                            "end": 4650
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4645,
                                          "end": 4650
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "isComplete",
                                          "loc": {
                                            "start": 4671,
                                            "end": 4681
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4671,
                                          "end": 4681
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminderItems",
                                          "loc": {
                                            "start": 4702,
                                            "end": 4715
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 4742,
                                                  "end": 4744
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4742,
                                                "end": 4744
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 4769,
                                                  "end": 4779
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4769,
                                                "end": 4779
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 4804,
                                                  "end": 4814
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4804,
                                                "end": 4814
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 4839,
                                                  "end": 4843
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4839,
                                                "end": 4843
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 4868,
                                                  "end": 4879
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4868,
                                                "end": 4879
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "dueDate",
                                                "loc": {
                                                  "start": 4904,
                                                  "end": 4911
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4904,
                                                "end": 4911
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 4936,
                                                  "end": 4941
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4936,
                                                "end": 4941
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isComplete",
                                                "loc": {
                                                  "start": 4966,
                                                  "end": 4976
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4966,
                                                "end": 4976
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 4716,
                                            "end": 4998
                                          }
                                        },
                                        "loc": {
                                          "start": 4702,
                                          "end": 4998
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 4453,
                                      "end": 5016
                                    }
                                  },
                                  "loc": {
                                    "start": 4443,
                                    "end": 5016
                                  }
                                }
                              ],
                              "loc": {
                                "start": 4352,
                                "end": 5030
                              }
                            },
                            "loc": {
                              "start": 4339,
                              "end": 5030
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "resourceList",
                              "loc": {
                                "start": 5043,
                                "end": 5055
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 5074,
                                      "end": 5076
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5074,
                                    "end": 5076
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 5093,
                                      "end": 5103
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5093,
                                    "end": 5103
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "translations",
                                    "loc": {
                                      "start": 5120,
                                      "end": 5132
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 5155,
                                            "end": 5157
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5155,
                                          "end": 5157
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "language",
                                          "loc": {
                                            "start": 5178,
                                            "end": 5186
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5178,
                                          "end": 5186
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 5207,
                                            "end": 5218
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5207,
                                          "end": 5218
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 5239,
                                            "end": 5243
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5239,
                                          "end": 5243
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 5133,
                                      "end": 5261
                                    }
                                  },
                                  "loc": {
                                    "start": 5120,
                                    "end": 5261
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "resources",
                                    "loc": {
                                      "start": 5278,
                                      "end": 5287
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 5310,
                                            "end": 5312
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5310,
                                          "end": 5312
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "index",
                                          "loc": {
                                            "start": 5333,
                                            "end": 5338
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5333,
                                          "end": 5338
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "link",
                                          "loc": {
                                            "start": 5359,
                                            "end": 5363
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5359,
                                          "end": 5363
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "usedFor",
                                          "loc": {
                                            "start": 5384,
                                            "end": 5391
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5384,
                                          "end": 5391
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 5412,
                                            "end": 5424
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 5451,
                                                  "end": 5453
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5451,
                                                "end": 5453
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 5478,
                                                  "end": 5486
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5478,
                                                "end": 5486
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 5511,
                                                  "end": 5522
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5511,
                                                "end": 5522
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 5547,
                                                  "end": 5551
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5547,
                                                "end": 5551
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5425,
                                            "end": 5573
                                          }
                                        },
                                        "loc": {
                                          "start": 5412,
                                          "end": 5573
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 5288,
                                      "end": 5591
                                    }
                                  },
                                  "loc": {
                                    "start": 5278,
                                    "end": 5591
                                  }
                                }
                              ],
                              "loc": {
                                "start": 5056,
                                "end": 5605
                              }
                            },
                            "loc": {
                              "start": 5043,
                              "end": 5605
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "schedule",
                              "loc": {
                                "start": 5618,
                                "end": 5626
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Schedule_common",
                                    "loc": {
                                      "start": 5648,
                                      "end": 5663
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 5645,
                                    "end": 5663
                                  }
                                }
                              ],
                              "loc": {
                                "start": 5627,
                                "end": 5677
                              }
                            },
                            "loc": {
                              "start": 5618,
                              "end": 5677
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "id",
                              "loc": {
                                "start": 5690,
                                "end": 5692
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5690,
                              "end": 5692
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "name",
                              "loc": {
                                "start": 5705,
                                "end": 5709
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5705,
                              "end": 5709
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "description",
                              "loc": {
                                "start": 5722,
                                "end": 5733
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5722,
                              "end": 5733
                            }
                          }
                        ],
                        "loc": {
                          "start": 3369,
                          "end": 5743
                        }
                      },
                      "loc": {
                        "start": 3358,
                        "end": 5743
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "handle",
                        "loc": {
                          "start": 5752,
                          "end": 5758
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5752,
                        "end": 5758
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasPremium",
                        "loc": {
                          "start": 5767,
                          "end": 5777
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5767,
                        "end": 5777
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "id",
                        "loc": {
                          "start": 5786,
                          "end": 5788
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5786,
                        "end": 5788
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "languages",
                        "loc": {
                          "start": 5797,
                          "end": 5806
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5797,
                        "end": 5806
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "membershipsCount",
                        "loc": {
                          "start": 5815,
                          "end": 5831
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5815,
                        "end": 5831
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "name",
                        "loc": {
                          "start": 5840,
                          "end": 5844
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5840,
                        "end": 5844
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "notesCount",
                        "loc": {
                          "start": 5853,
                          "end": 5863
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5853,
                        "end": 5863
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "projectsCount",
                        "loc": {
                          "start": 5872,
                          "end": 5885
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5872,
                        "end": 5885
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "questionsAskedCount",
                        "loc": {
                          "start": 5894,
                          "end": 5913
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5894,
                        "end": 5913
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "routinesCount",
                        "loc": {
                          "start": 5922,
                          "end": 5935
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5922,
                        "end": 5935
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "smartContractsCount",
                        "loc": {
                          "start": 5944,
                          "end": 5963
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5944,
                        "end": 5963
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "standardsCount",
                        "loc": {
                          "start": 5972,
                          "end": 5986
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5972,
                        "end": 5986
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "theme",
                        "loc": {
                          "start": 5995,
                          "end": 6000
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5995,
                        "end": 6000
                      }
                    }
                  ],
                  "loc": {
                    "start": 351,
                    "end": 6006
                  }
                },
                "loc": {
                  "start": 345,
                  "end": 6006
                }
              }
            ],
            "loc": {
              "start": 311,
              "end": 6010
            }
          },
          "loc": {
            "start": 300,
            "end": 6010
          }
        }
      ],
      "loc": {
        "start": 296,
        "end": 6012
      }
    },
    "loc": {
      "start": 276,
      "end": 6012
    }
  },
  "variableValues": {},
  "path": {
    "key": "auth_guestLogIn"
  }
} as const;
